package lsp

import "time"

type ServerConfig struct {
	Language       Language      `yaml:"language" json:"language"`
	Command        string        `yaml:"command" json:"command"`
	Args           []string      `yaml:"args,omitempty" json:"args,omitempty"`
	RootPatterns   []string      `yaml:"root_patterns" json:"root_patterns"`
	Extensions     []string      `yaml:"extensions" json:"extensions"`
	Enabled        bool          `yaml:"enabled" json:"enabled"`
	InitTimeout    time.Duration `yaml:"init_timeout" json:"init_timeout"`
	RequestTimeout time.Duration `yaml:"request_timeout" json:"request_timeout"`
	MaxRestarts    int           `yaml:"max_restarts" json:"max_restarts"`
}

type ManagerConfig struct {
	Enabled        bool                      `yaml:"enabled" json:"enabled"`
	AutoStart      bool                      `yaml:"auto_start" json:"auto_start"`
	IdleTimeout    time.Duration             `yaml:"idle_timeout" json:"idle_timeout"`
	RequestTimeout time.Duration             `yaml:"request_timeout" json:"request_timeout"`
	MaxConcurrent  int                       `yaml:"max_concurrent" json:"max_concurrent"`
	Servers        map[Language]ServerConfig `yaml:"servers" json:"servers"`
}

func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		Enabled:        true,
		AutoStart:      false,
		IdleTimeout:    10 * time.Minute,
		RequestTimeout: 30 * time.Second,
		MaxConcurrent:  3,
		Servers: map[Language]ServerConfig{
			LangGo: {
				Language:       LangGo,
				Command:        "gopls",
				Args:           []string{"serve"},
				RootPatterns:   []string{"go.mod", "go.work"},
				Extensions:     []string{".go"},
				Enabled:        true,
				InitTimeout:    10 * time.Second,
				RequestTimeout: 30 * time.Second,
				MaxRestarts:    3,
			},
			LangTypeScript: {
				Language:       LangTypeScript,
				Command:        "typescript-language-server",
				Args:           []string{"--stdio"},
				RootPatterns:   []string{"package.json", "tsconfig.json"},
				Extensions:     []string{".ts", ".tsx"},
				Enabled:        true,
				InitTimeout:    15 * time.Second,
				RequestTimeout: 30 * time.Second,
				MaxRestarts:    3,
			},
			LangJavaScript: {
				Language:       LangJavaScript,
				Command:        "typescript-language-server",
				Args:           []string{"--stdio"},
				RootPatterns:   []string{"package.json"},
				Extensions:     []string{".js", ".jsx", ".mjs"},
				Enabled:        true,
				InitTimeout:    15 * time.Second,
				RequestTimeout: 30 * time.Second,
				MaxRestarts:    3,
			},
			LangPython: {
				Language:       LangPython,
				Command:        "pylsp",
				Args:           []string{},
				RootPatterns:   []string{"pyproject.toml", "setup.py", "requirements.txt"},
				Extensions:     []string{".py"},
				Enabled:        true,
				InitTimeout:    10 * time.Second,
				RequestTimeout: 30 * time.Second,
				MaxRestarts:    3,
			},
			LangRust: {
				Language:       LangRust,
				Command:        "rust-analyzer",
				Args:           []string{},
				RootPatterns:   []string{"Cargo.toml"},
				Extensions:     []string{".rs"},
				Enabled:        true,
				InitTimeout:    20 * time.Second,
				RequestTimeout: 30 * time.Second,
				MaxRestarts:    2,
			},
			LangCpp: {
				Language:       LangCpp,
				Command:        "clangd",
				Args:           []string{},
				RootPatterns:   []string{"compile_commands.json", "CMakeLists.txt", "Makefile"},
				Extensions:     []string{".cpp", ".cc", ".cxx", ".hpp", ".h"},
				Enabled:        true,
				InitTimeout:    10 * time.Second,
				RequestTimeout: 30 * time.Second,
				MaxRestarts:    3,
			},
			LangC: {
				Language:       LangC,
				Command:        "clangd",
				Args:           []string{},
				RootPatterns:   []string{"compile_commands.json", "Makefile"},
				Extensions:     []string{".c", ".h"},
				Enabled:        true,
				InitTimeout:    10 * time.Second,
				RequestTimeout: 30 * time.Second,
				MaxRestarts:    3,
			},
			LangJava: {
				Language:       LangJava,
				Command:        "jdtls",
				Args:           []string{},
				RootPatterns:   []string{"pom.xml", "build.gradle", "settings.gradle"},
				Extensions:     []string{".java"},
				Enabled:        false,
				InitTimeout:    30 * time.Second,
				RequestTimeout: 30 * time.Second,
				MaxRestarts:    2,
			},
		},
	}
}

func (c *ManagerConfig) GetServerForExtension(ext string) (ServerConfig, bool) {
	for _, server := range c.Servers {
		if !server.Enabled {
			continue
		}
		for _, e := range server.Extensions {
			if e == ext {
				return server, true
			}
		}
	}
	return ServerConfig{}, false
}

func (c *ManagerConfig) GetEnabledLanguages() []Language {
	var langs []Language
	for lang, server := range c.Servers {
		if server.Enabled {
			langs = append(langs, lang)
		}
	}
	return langs
}
