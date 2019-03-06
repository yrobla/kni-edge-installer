package generator

import (
    "fmt"
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
    binaryPath := fmt.Sprintf("%s/openshift-install", g.buildPath)
    client := &getter.Client{
        Src: g.installerPath,
        Dst: binaryPath,
    }
    err := client.Get()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    os.Chmod(binaryPath, 0777)
}
