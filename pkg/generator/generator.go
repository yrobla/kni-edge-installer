package generator

import (
    "encoding/base64"
    "fmt"
    "io/ioutil"
    "log"
    "os"

    getter "github.com/hashicorp/go-getter"
)

type generator struct {
    baseRepo       string
    basePath       string
    installerPath  string
    secretsRepo    string
    settingsPath   string
    buildPath      string
}

func New(baseRepo string, basePath string, installerPath string, secretsRepo string, settingsPath string, buildPath string) generator {
    g := generator {baseRepo, basePath, installerPath, secretsRepo, settingsPath, buildPath}
    return g
}

func (g generator) GenerateFromInstall() {
    // Download installer for openshift
    log.Println("Downloading openshift-install binary")
    binaryPath := fmt.Sprintf("%s/openshift-install", g.buildPath)
    client := &getter.Client{Src: g.installerPath, Dst: binaryPath}
    err := client.Get()
    if err != nil {
        log.Fatal(fmt.Sprintf("Error downloading openshift-install binary: %s", err))
        os.Exit(1)
    }
    os.Chmod(binaryPath, 0777)

    // Download the credentials repo
    log.Println("Download secrets repo")
    secretsPath := fmt.Sprintf("%s/secrets", g.buildPath)

    // Retrieve private key and b64encode it
    rsaPrivateLocation := fmt.Sprintf("%s/.ssh/id_rsa", os.Getenv("HOME"))
    priv, _ := ioutil.ReadFile(rsaPrivateLocation)
    sEnc := base64.StdEncoding.EncodeToString(priv)

    finalUrl := fmt.Sprintf("%s?sshkey=%s", g.secretsRepo, sEnc)
    log.Println(finalUrl)
    client = &getter.Client{Src: finalUrl, Dst: secretsPath, Mode: getter.ClientModeAny}
    err = client.Get()
    if err != nil {
        log.Println(secretsPath)
        log.Fatal(fmt.Sprintf("Error downloading secrets repo: %s", err))
        os.Exit(1)
    }
    os.Chmod(secretsPath, 0700)
}
