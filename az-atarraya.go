package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/joho/godotenv"
)

var (
	envprefix        = "ATARRAYA_"
	secretprefix     = envprefix + "SECRET_"
	keyvaultName     string
	keyvaultEndpoint string
)

func main() {
	godotenv.Load()

	var secretsToRead []string
	for _, varName := range os.Environ() {
		if strings.HasPrefix(varName, secretprefix) {
			pair := strings.SplitN(varName, "=", 2)
			secretsToRead = append(secretsToRead, strings.Replace(pair[0], secretprefix, "", -1))
		}
	}

	// Get the Key Vault name from the Environment
	keyvaultName = os.Getenv(envprefix + "AZURE_KEYVAULT_NAME")
	keyvaultEndpoint = fmt.Sprintf("https://%s.vault.azure.net", keyvaultName)

	for _, secretName := range secretsToRead {
		// Fetch secret here & append it to the environment vars
		secret, err := getKeyVaultSecret(secretName, "")
		if err != nil {
			log.Fatal(err)
		}
		os.Setenv(secretName, secret)
	}

	commandToRun := flag.String("c", "", "command to run")
	commandArgs := flag.String("args", "", "command arguments")
	flag.Parse()

	fmt.Println("Command to run:", *commandToRun)
	fmt.Println("Arguments:", *commandArgs)

	cmd := exec.Command(*commandToRun, *commandArgs)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func getKeyVaultSecret(secretname string, secretversion string) (string, error) {
	keyvaultSecretName := secretname
	keyvaultSecretVersion := secretversion
	secretVersionPresent := len(secretversion) > 0

	// Create the key vault client & authorizer
	keyVaultClient := keyvault.New()
	authorizer, err := auth.NewAuthorizerFromEnvironment()

	if err == nil {
		keyVaultClient.Authorizer = authorizer
	}

	if !secretVersionPresent {
		result, err := keyVaultClient.GetSecretVersions(context.Background(), keyvaultEndpoint, keyvaultSecretName, nil)

		if err != nil {
			log.Printf("failed to retrieve Keyvault secret versions: %v", err)
			return "", err
		}

		var secretDate time.Time
		var secretVersion string
		for result.NotDone() {
			for _, secret := range result.Values() {
				if *secret.Attributes.Enabled {
					updatedTime := time.Time(*secret.Attributes.Updated)
					if secretDate.IsZero() || updatedTime.After(secretDate) {
						secretDate = updatedTime

						// Get the version
						parts := strings.Split(*secret.ID, "/")
						secretVersion = parts[len(parts)-1]
					}
				}
			}

			result.Next()
		}
		keyvaultSecretVersion = secretVersion
	}

	log.Printf("reading secret %s with version %s", keyvaultSecretName, keyvaultSecretVersion)

	// Get and return the secret
	secret, err := keyVaultClient.GetSecret(context.Background(), keyvaultEndpoint, keyvaultSecretName, keyvaultSecretVersion)
	if err != nil {
		log.Printf("failed to retrieve the Keyvault secret: %v", err)
		return "", err
	}

	log.Printf("secret %s with version %s was found and returned", keyvaultSecretName, keyvaultSecretVersion)
	return *secret.Value, nil
}
