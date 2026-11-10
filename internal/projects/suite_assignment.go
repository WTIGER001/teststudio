package projects

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) validateSuiteAssignment(ctx context.Context, project string, suite Suite) error {
	application := suite.Applications[0]
	rows, err := s.Store.All(ctx, "tests/"+project+"/"+suite.ID, project)
	if err != nil {
		return err
	}
	for _, row := range rows {
		var test struct {
			Application string `json:"applicationId"`
			State       string `json:"state"`
		}
		if err := json.Unmarshal(row.Data, &test); err != nil {
			return err
		}
		if test.State != "archived" && test.Application != "" && test.Application != application {
			return fmt.Errorf("a current test belongs to another application; resolve it before assigning this suite")
		}
	}
	for _, bucket := range []string{"schedules/" + project, "probes/" + project} {
		rows, err := s.Store.All(ctx, bucket, project)
		if err != nil {
			return err
		}
		for _, row := range rows {
			var value struct {
				Archived    bool   `json:"archived"`
				Suite       string `json:"suiteId"`
				Environment string `json:"environmentId"`
				Request     struct {
					Suite       string `json:"suiteId"`
					Application string `json:"applicationId"`
					Environment string `json:"environmentId"`
				} `json:"request"`
			}
			if err := json.Unmarshal(row.Data, &value); err != nil {
				return err
			}
			if value.Archived || value.Suite != suite.ID && value.Request.Suite != suite.ID {
				continue
			}
			if value.Request.Application != "" && value.Request.Application != application {
				return fmt.Errorf("a current resource selects another application; update it before assigning this suite")
			}
			environment := value.Environment
			if environment == "" {
				environment = value.Request.Environment
			}
			if environment == "" {
				continue
			}
			target, err := s.Store.Get(ctx, Bucket("environments", project), environment)
			if err != nil {
				return err
			}
			e, err := storage.Decode[Environment](target)
			if err != nil {
				return err
			}
			if e.Application != application {
				return fmt.Errorf("a current resource selects another application's environment; update it before assigning this suite")
			}
		}
	}
	return nil
}
