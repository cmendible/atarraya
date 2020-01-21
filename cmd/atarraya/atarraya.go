package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/joho/godotenv"
)

var (
	envprefix    = "ATARRAYA_"
	secretprefix = envprefix + "SECRET_"
)

type kvClient struct {
	Instance         keyvault.BaseClient
	keyvaultName     string
	keyvaultEndpoint string
}

func newkvClient(keyvaultName string) *kvClient {
	client := new(kvClient)
	client.keyvaultName = keyvaultName
	client.keyvaultEndpoint = fmt.Sprintf("https://%s.vault.azure.net", keyvaultName)

	log.Printf("keyvaultEndpoint: %s", client.keyvaultEndpoint)

	client.Instance = keyvault.New()

	// Internals: This creates an Authorizer configured from environment variables in the following order:
	// 1. Client credentials
	// 2. Client certificate
	// 3. Username password
	// 4. MSI
	authorizer, err := auth.NewAuthorizerFromEnvironment()

	if err == nil {
		client.Instance.Authorizer = authorizer
	}

	return client
}

func (kv *kvClient) getKeyVaultSecret(secretname string) (string, error) {
	keyvaultSecretName := secretname

	log.Printf("reading secret %s", keyvaultSecretName)

	// Get and return the secret
	secret, err := kv.Instance.GetSecret(context.Background(), kv.keyvaultEndpoint, keyvaultSecretName, "")
	if err != nil {
		log.Printf("failed to retrieve the Keyvault secret: %v", err)
		return "", err
	}

	log.Printf("secret %s was found and returned", keyvaultSecretName)
	return *secret.Value, nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Print(err)
	}

	var cmd []string
	if len(os.Args) == 1 {
		log.Fatalln("no command is given, atarraya can't determine the entrypoint (command), please specify it explicitly or let the webhook query it (see documentation)")
	} else {
		cmd = os.Args[1:]
	}

	_, err := exec.LookPath(cmd[0])
	if err != nil {
		log.Fatalln("binary not found", cmd[0])
	}

	var secretsToRead []string
	for _, varName := range os.Environ() {
		if strings.HasPrefix(varName, secretprefix) {
			pair := strings.SplitN(varName, "=", 2)
			secretsToRead = append(secretsToRead, strings.Replace(pair[0], secretprefix, "", -1))
		}
	}

	var client = newkvClient(os.Getenv(envprefix + "AZURE_KEYVAULT_NAME"))

	for _, secretName := range secretsToRead {
		// Fetch secret here & append it to the environment vars
		log.Printf("Reading: %s", secretName)
		secret, err := client.getKeyVaultSecret(secretName)
		if err != nil {
			log.Fatal(err)
		}
		os.Setenv(secretName, secret)
	}

	executable := exec.Command(cmd[0], cmd[1:]...)
	executable.Env = os.Environ()
	executable.Stdout = os.Stdout
	executable.Stderr = os.Stderr
	err = executable.Run()
	if err != nil {
		log.Fatal(err)
	}
}
