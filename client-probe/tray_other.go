//go:build !windows || cli

package main

import "context"

func initSystemTray(ctx context.Context, app interface{}) {}
func showTrayBalloon(title, message string)               {}
func cleanupTray()                                        {}
