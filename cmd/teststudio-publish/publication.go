package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

func publication(ctx context.Context, operation string, args []string, writer io.Writer) error {
	flags := flag.NewFlagSet(operation, flag.ContinueOnError)
	flags.SetOutput(writer)
	session := flags.String("session-file", sessionPath(), "Private managed-session file")
	project := flags.String("project", "", "Destination project identifier")
	suite := flags.String("suite", "", "Destination suite identifier")
	jobID := flags.String("job", "", "Preparation identifier")
	image := flags.String("image", "", "Immutable image to prepare")
	runner := flags.String("runner", "go", "go or playwright")
	key := flags.String("key", "", "Idempotency key for prepare retries")
	activate := flags.Bool("activate", false, "Activate the published release for future runs")
	review := flags.Bool("review", false, "Show the prepared catalog changes")
	logs := flags.Bool("logs", false, "Show the retained private preparation log")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || !domain.ValidID(*project) || !domain.ValidID(*suite) {
		return errors.New("project and suite identifiers are required")
	}
	client, err := readClient(*session)
	if err != nil {
		return err
	}
	base := "/api/v1/projects/" + url.PathEscape(*project) + "/suites/" + url.PathEscape(*suite)
	var row storage.Record
	if operation == "prepare" {
		if *image == "" || (*runner != "go" && *runner != "playwright") {
			return errors.New("prepare requires an immutable image and supported runner")
		}
		if *key == "" {
			var data [16]byte
			if _, err := rand.Read(data[:]); err != nil {
				return err
			}
			*key = hex.EncodeToString(data[:])
		}
		fmt.Fprintln(writer, "Preparation retry key:", *key)
		if err := client.request(ctx, "POST", base+"/source-preparations", sources.PrebuiltInput{Image: *image, Runner: *runner}, 0, *key, &row); err != nil {
			return err
		}
	} else {
		if !domain.ValidID(*jobID) {
			return errors.New("a preparation identifier is required")
		}
		if err := client.request(ctx, "GET", base+"/source-preparations/"+url.PathEscape(*jobID), nil, 0, "", &row); err != nil {
			return err
		}
	}
	job, err := storage.Decode[sources.Job](row)
	if err != nil {
		return err
	}
	if operation == "cancel" {
		if err := client.request(ctx, "POST", base+"/source-preparations/"+url.PathEscape(job.ID)+"/cancel", map[string]any{}, row.Revision, "", &row); err != nil {
			return err
		}
		job, err = storage.Decode[sources.Job](row)
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(writer, "Preparation %s: %s (%s)\n", job.ID, job.State, job.Phase)
	if job.Error != "" {
		fmt.Fprintln(writer, job.Error)
	}
	if *logs && job.Log != "" {
		fmt.Fprintln(writer, job.Log)
		if job.LogTruncated {
			fmt.Fprintln(writer, "Preparation log is truncated.")
		}
	}
	if job.Upload == "" {
		if operation == "publish" {
			return errors.New("preparation has no ready source review")
		}
		return nil
	}
	fmt.Fprintln(writer, "Source review:", job.Upload)
	if operation != "publish" && !*review {
		return nil
	}
	var preview sources.Preview
	if err := client.request(ctx, "GET", base+"/source-uploads/"+url.PathEscape(job.Upload), nil, 0, "", &preview); err != nil {
		return err
	}
	upload, err := storage.Decode[sources.Upload](preview.Upload)
	if err != nil {
		return err
	}
	fmt.Fprintln(writer, "Stored source digest:", upload.Digest)
	if preview.Plan != nil {
		for _, change := range preview.Plan.Changes {
			fmt.Fprintf(writer, "%s %s: %s [%s, %s]\n", change.Action, change.Test.ID, change.Test.Title, change.Test.Type, change.Test.Safety)
		}
		for _, issue := range preview.Plan.Issues {
			fmt.Fprintf(writer, "Validation: %s %s: %s\n", issue.TestID, issue.Field, issue.Message)
		}
	}
	if operation != "publish" {
		return nil
	}
	if job.State != "ready" && job.State != "published" {
		return errors.New("preparation is not ready for publication")
	}
	var release catalog.Release
	if err := client.request(ctx, "POST", base+"/source-uploads/"+url.PathEscape(job.Upload)+"/publish", map[string]bool{"activate": *activate}, preview.Upload.Revision, "", &release); err != nil {
		return err
	}
	fmt.Fprintf(writer, "Published release %s; activated: %t. No tests were started.\n", release.ID, release.ActivatedOnPublication)
	return nil
}
