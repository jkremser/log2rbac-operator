package internal

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// SetupLog tweak the default log to use custom time format and use colors if supported
func SetupLog(cfg *LogConfig) {
	var opts zap.Options
	zap.UseDevMode(true)(&opts)
	zap.ConsoleEncoder(func(c *zapcore.EncoderConfig) {
		c.EncodeTime = zapcore.TimeEncoderOfLayout("01-02 15:04:05")
		if cfg.Colors {
			c.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
	})(&opts)
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	// todo: use wrapper and add the context that called the logger (https://stackoverflow.com/questions/61246838/zap-logger-source-line)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
}

// PrintBanner prints the ascii banner for log2rbac app in logs
// the fatih/color module should be smart enough to recognize the attached
// stdout's file descriptor if it's capable of colors, but we can explicitly control this
// by cfg.Colors bool
func PrintBanner(cfg *LogConfig) {
	if cfg.NoBanner {
		return
	}
	c1, c2, c3 := color.FgCyan, color.FgHiWhite, color.FgBlue
	if !cfg.Colors {
		c1, c2, c3 = color.Reset, color.Reset, color.Reset
	}
	pad := "     " + "     " + "     " + "     " + "     "
	lines := []string{
		pad + color.New(c1).Sprintf(" _             ") + color.New(c2).Sprintf("____       ") + color.New(c3).Sprintf("_"),
		pad + color.New(c1).Sprintf("| | ___   __ _") + color.New(c2).Sprintf("|___ \\") + color.New(c3).Sprintf(" _ __| |__   __ _  ___"),
		pad + color.New(c1).Sprintf("| |/ _ \\ / _` | ") + color.New(c2).Sprintf("__) ") + color.New(c3).Sprintf("| '__| '_ \\ / _` |/ __|"),
		pad + color.New(c1).Sprintf("| | (_) | (_| |") + color.New(c2).Sprintf("/ __/") + color.New(c3).Sprintf("| |  | |_) | (_| | (__"),
		pad + color.New(c1).Sprintf("|_|\\___/ \\__, ") + color.New(c2).Sprintf("|_____") + color.New(c3).Sprintf("|_|  |_.__/ \\__,_|\\___|"),
		pad + color.New(c1).Sprintf("         |___/"),
	}
	for _, line := range lines {
		fmt.Println(line)
	}
}
