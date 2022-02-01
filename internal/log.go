package internal

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func SetupLog() {
	var opts zap.Options
	zap.UseDevMode(true)(&opts)
	zap.ConsoleEncoder(func(c *zapcore.EncoderConfig) {
		c.EncodeTime = zapcore.TimeEncoderOfLayout("01-02 15:04:05")
		c.EncodeLevel = zapcore.CapitalColorLevelEncoder
	})(&opts)
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
}

func PrintBanner() {
	c1 := color.FgCyan
	c2 := color.FgHiWhite
	c3 := color.FgBlue
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
