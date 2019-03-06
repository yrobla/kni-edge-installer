package generator

import (
    "fmt"
)

type generator struct {
    baseRepo       string
    basePath       string
    installerPath  string
    secretsRepo    string
    settingsPath   string
}

func New(baseRepo string, basePath string, installerPath string, secretsRepo string, settingsPath string) generator {
    g := generator {baseRepo, basePath, installerPath, secretsRepo, settingsPath}
    return g
}

func (g generator) GenerateFromInstall() {
    fmt.Println("here")
    fmt.Println(g.baseRepo)
}
