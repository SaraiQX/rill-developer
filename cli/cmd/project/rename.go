package project

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/rilldata/rill/cli/cmd/cmdutil"
	"github.com/rilldata/rill/cli/pkg/config"
	adminv1 "github.com/rilldata/rill/proto/gen/rill/admin/v1"
	"github.com/spf13/cobra"
)

func RenameCmd(cfg *config.Config) *cobra.Command {
	var name, newName string

	renameCmd := &cobra.Command{
		Use:   "rename",
		Args:  cobra.NoArgs,
		Short: "Rename",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := cmdutil.Client(cfg)
			if err != nil {
				return err
			}
			defer client.Close()

			if !cmd.Flags().Changed("project") {
				resp, err := client.ListProjectsForOrganization(ctx, &adminv1.ListProjectsForOrganizationRequest{OrganizationName: cfg.Org})
				if err != nil {
					return err
				}

				if len(resp.Projects) == 0 {
					return fmt.Errorf("No projects found for org %q", cfg.Org)
				}

				var projectNames []string
				for _, proj := range resp.Projects {
					projectNames = append(projectNames, proj.Name)
				}

				name = cmdutil.SelectPrompt("Select project to rename", projectNames, "")
			}

			if !cmd.Flags().Changed("new_name") {
				// Get the new project name from user if not provided in the flag, passing current name as default
				newName, err = cmdutil.InputPrompt("Rename to", name)
				if err != nil {
					return err
				}
			}

			fmt.Println("Warn: Renaming an project would invalidate dashboard URLs")

			msg := fmt.Sprintf("Do you want to rename project \"%s\" to \"%s\"?", color.YellowString(name), color.YellowString(newName))
			if !cmdutil.ConfirmPrompt(msg, "", false) {
				return nil
			}

			resp, err := client.GetProject(ctx, &adminv1.GetProjectRequest{OrganizationName: cfg.Org, Name: name})
			if err != nil {
				return err
			}

			proj := resp.Project

			updatedProj, err := client.UpdateProject(ctx, &adminv1.UpdateProjectRequest{
				Id:               proj.Id,
				OrganizationName: cfg.Org,
				Name:             newName,
				Description:      proj.Description,
				Public:           proj.Public,
				ProdBranch:       proj.ProdBranch,
				GithubUrl:        proj.GithubUrl,
			})
			if err != nil {
				return err
			}

			cmdutil.SuccessPrinter("Renamed project \n")
			cmdutil.TablePrinter(toRow(updatedProj.Project))

			return nil
		},
	}

	renameCmd.Flags().SortFlags = false
	renameCmd.Flags().StringVar(&name, "project", "", "Current Project Name")
	renameCmd.Flags().StringVar(&newName, "new_name", "", "New Project Name")

	return renameCmd
}
