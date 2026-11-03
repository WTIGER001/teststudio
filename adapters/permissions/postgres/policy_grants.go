package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
)

const grantColumns = `id,owner_kind,owner_id,effect,team_scope,object_scope,permission_name,expires_at,to_json(field_allowlist),variable_spec`

func scanGrant(row interface{ Scan(...any) error }) (permissions.Grant, error) {
	var grant permissions.Grant
	var fields, variables []byte
	if err := row.Scan(&grant.ID, &grant.OwnerKind, &grant.OwnerID, &grant.Effect, &grant.TeamScope, &grant.ObjectScope, &grant.PermissionName, &grant.ExpiresAt, &fields, &variables); err != nil {
		return grant, err
	}
	if len(fields) > 0 {
		if err := json.Unmarshal(fields, &grant.RestrictedFields); err != nil {
			return grant, err
		}
	}
	if err := json.Unmarshal(variables, &grant.VariableSpec); err != nil {
		return grant, err
	}
	return grant, nil
}

func validatePolicyGrant(grant permissions.Grant) error {
	if err := (permissions.PrincipalRef{Kind: grant.OwnerKind, ID: grant.OwnerID}).Validate(); err != nil {
		return err
	}
	if grant.OwnerKind == permissions.PrincipalTeam {
		return errors.New("teams receive roles rather than direct grants")
	}
	if grant.OwnerKind == permissions.PrincipalRole && strings.HasPrefix(grant.OwnerID, "builtin.") {
		return errors.New("built-in role grants are immutable")
	}
	if grant.TeamScope == "" || grant.PermissionName == "" {
		return errors.New("permission and scope are required")
	}
	if grant.Effect != permissions.EffectAllow && grant.Effect != permissions.EffectDeny {
		return errors.New("invalid grant effect")
	}
	return nil
}

func (s *policyTransaction) CreateGrant(ctx context.Context, grant permissions.Grant) error {
	if err := validatePolicyGrant(grant); err != nil {
		return err
	}
	fields, err := json.Marshal(grant.RestrictedFields)
	if err != nil {
		return err
	}
	variables, err := json.Marshal(grant.VariableSpec)
	if err != nil {
		return err
	}
	if grant.VariableSpec == nil {
		variables = []byte("{}")
	}
	_, err = s.tx.ExecContext(ctx, `INSERT INTO teststudio_permissions.permission_grants (owner_kind,owner_id,effect,team_scope,object_scope,permission_name,expires_at,field_allowlist,variable_spec)
VALUES ($1,$2,$3,$4,$5,$6,$7,CASE WHEN $8::jsonb='null'::jsonb THEN NULL ELSE ARRAY(SELECT jsonb_array_elements_text($8::jsonb)) END,$9::jsonb)`, string(grant.OwnerKind), grant.OwnerID, string(grant.Effect), grant.TeamScope, grant.ObjectScope, grant.PermissionName, grant.ExpiresAt, string(fields), string(variables))
	return err
}

func (s *policyTransaction) CreateGrants(ctx context.Context, grants []permissions.Grant) error {
	if len(grants) > 100 {
		return errors.New("maximum 100 grants per transaction")
	}
	for _, g := range grants {
		if err := validatePolicyGrant(g); err != nil {
			return err
		}
	}
	for _, g := range grants {
		if err := s.CreateGrant(ctx, g); err != nil {
			return err
		}
	}
	return nil
}

func (s *policyTransaction) DeleteGrantsForOwner(ctx context.Context, kind permissions.PrincipalKind, id string) error {
	if err := (permissions.PrincipalRef{Kind: kind, ID: id}).Validate(); err != nil {
		return err
	}
	if kind == permissions.PrincipalRole && strings.HasPrefix(id, "builtin.") {
		return errors.New("built-in role grants are immutable")
	}
	_, err := s.tx.ExecContext(ctx, `DELETE FROM teststudio_permissions.permission_grants WHERE owner_kind=$1 AND owner_id=$2`, string(kind), id)
	return err
}

func (s *policyTransaction) RemoveGrant(ctx context.Context, id int64) error {
	if id < 1 {
		return errors.New("a positive grant identifier is required")
	}
	_, err := s.tx.ExecContext(ctx, `DELETE FROM teststudio_permissions.permission_grants WHERE id=$1`, id)
	return err
}

func placeholder(n int) string { return fmt.Sprintf("$%d", n) }

func grantFilter(query permissions.GrantQuery) (string, []any) {
	conditions := []string{}
	args := []any{}
	add := func(v any) string { args = append(args, v); return placeholder(len(args)) }
	if len(query.Principals) > 0 {
		owners := []string{}
		for _, p := range query.Principals {
			owners = append(owners, `(owner_kind=`+add(string(p.Kind))+` AND owner_id=`+add(p.ID)+`)`)
		}
		conditions = append(conditions, `(`+strings.Join(owners, " OR ")+`)`)
	}
	for _, entry := range []struct {
		column string
		values []string
	}{{"team_scope", query.TeamScopes}, {"object_scope", query.ObjectScopes}, {"permission_name", query.Permissions}} {
		if len(entry.values) == 0 {
			continue
		}
		marks := []string{}
		for _, v := range entry.values {
			marks = append(marks, add(v))
		}
		conditions = append(conditions, entry.column+` IN (`+strings.Join(marks, ",")+`)`)
	}
	if query.PermissionPrefix != "" {
		mark := add(query.PermissionPrefix)
		conditions = append(conditions, `left(permission_name,length(`+mark+`::text))=`+mark)
	}
	if len(conditions) == 0 {
		return "true", args
	}
	return strings.Join(conditions, " AND "), args
}

func (s *policyTransaction) GrantsForPrincipal(ctx context.Context, p permissions.PrincipalRef) ([]permissions.Grant, error) {
	return s.GrantsForOwners(ctx, []permissions.PrincipalRef{p}, permissions.Request{})
}

func (s *policyTransaction) GrantsForOwners(ctx context.Context, owners []permissions.PrincipalRef, req permissions.Request) ([]permissions.Grant, error) {
	if len(owners) == 0 {
		return []permissions.Grant{}, nil
	}
	for _, p := range owners {
		if err := p.Validate(); err != nil {
			return nil, err
		}
	}
	q := permissions.GrantQuery{Principals: owners}
	if req.Perm != "" {
		q.Permissions = []string{req.Perm}
	}
	where, args := grantFilter(q)
	rows, err := s.tx.QueryContext(ctx, `SELECT `+grantColumns+` FROM teststudio_permissions.permission_grants WHERE `+where+` AND (expires_at IS NULL OR expires_at>now()) ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []permissions.Grant{}
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

func (s *policyTransaction) PrincipalsWithGrant(ctx context.Context, req permissions.Request) ([]permissions.PrincipalHit, error) {
	if req.Perm == "" {
		return nil, errors.New("permission is required")
	}
	rows, err := s.tx.QueryContext(ctx, `SELECT `+grantColumns+` FROM teststudio_permissions.permission_grants WHERE permission_name=$1 AND team_scope IN ('*',$2) AND (object_scope IS NULL OR object_scope IN ('*',$3)) AND (expires_at IS NULL OR expires_at>now()) ORDER BY id`, req.Perm, req.TeamID, req.Object)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	grants := []permissions.Grant{}
	denied := map[permissions.PrincipalRef]bool{}
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		grants = append(grants, g)
		if g.Effect == permissions.EffectDeny {
			denied[permissions.PrincipalRef{Kind: g.OwnerKind, ID: g.OwnerID}] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := []permissions.PrincipalHit{}
	seen := map[string]bool{}
	for _, g := range grants {
		if g.Effect != permissions.EffectAllow || denied[permissions.PrincipalRef{Kind: g.OwnerKind, ID: g.OwnerID}] {
			continue
		}
		hit := permissions.PrincipalHit{Kind: g.OwnerKind, ID: g.OwnerID, TeamScope: g.TeamScope, ObjectScope: g.ObjectScope, PermissionName: g.PermissionName}
		raw, _ := json.Marshal(hit)
		key := string(raw)
		if !seen[key] {
			seen[key] = true
			result = append(result, hit)
		}
	}
	return result, nil
}
