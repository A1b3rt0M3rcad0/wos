// An independent remote consumer needs only an endpoint, credential and Namespace.
package main

import (
	"context"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	wossdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	client, err := wossdk.New(os.Getenv("WOS_URL"), os.Getenv("WOS_TOKEN"), nil)
	if err != nil {
		return err
	}
	namespace, err := domain.ParseID(os.Getenv("WOS_NAMESPACE_ID"))
	if err != nil {
		return err
	}
	ctx := context.Background()
	cursor := ""
	for {
		page, err := client.SearchOutcomes(ctx, namespace, "", 25, cursor)
		if err != nil {
			return err
		}
		for _, outcome := range page.Items {
			snapshot, err := client.GetContinuity(ctx, outcome.Scope(), 10)
			if err != nil {
				return err
			}
			fmt.Printf("%s [%s], revision %d, omitted %v\n", snapshot.Outcome.Title, snapshot.Outcome.Lifecycle, snapshot.OutcomeRevision, snapshot.Omitted)
		}
		if page.NextCursor == "" {
			return nil
		}
		cursor = page.NextCursor
	}
}
