package serviceaccounts

import (
	"context"
	"encoding/json"
	flags2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/flags"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// NewDeleteCommand command for deleting kafkas.
func NewResetCredsCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-credentials",
		Short: "Reset Credentials of a serviceaccount",
		Long:  "Reset Credentials of a serviceaccount.",
		Run: func(cmd *cobra.Command, args []string) {
			runResetCreds(env, cmd)
		},
	}

	cmd.Flags().String(FlagSaID, "", "Service Account id")
	cmd.Flags().String(FlagOrgID, "", "OCM org id")
	return cmd
}

func runResetCreds(env *environments.Env, cmd *cobra.Command) {
	id := flags2.MustGetDefinedString(FlagSaID, cmd.Flags())
	orgId := flags2.MustGetDefinedString(FlagOrgID, cmd.Flags())

	// setup required services
	keycloakService := services.NewKeycloakService(env.Config.Keycloak, env.Config.Keycloak.KafkaRealm)

	// create jwt with claims and set it in the context
	jwt := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"org_id": orgId,
	})
	ctx := auth.SetTokenInContext(context.TODO(), jwt)

	sa, err := keycloakService.ResetServiceAccountCredentials(ctx, id)
	if err != nil {
		glog.Fatalf("Unable to reset credentials of a service account: %s", err.Error())
	}
	output, marshalErr := json.MarshalIndent(sa, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format service account: %s", err.Error())
	}
	glog.V(10).Infof("%s", output)
}