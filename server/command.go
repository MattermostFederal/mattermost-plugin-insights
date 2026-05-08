package main

import (
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const commandTrigger = "insights"

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          commandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: "Mattermost Insights commands",
		AutoCompleteHint: "[command]",
		DisplayName:      "Mattermost Insights",
	}
}

func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	fields := strings.Fields(args.Command)
	if len(fields) < 2 {
		return ephemeralResponse("Available subcommands: help"), nil
	}

	switch fields[1] {
	case "help":
		return ephemeralResponse("Mattermost Insights surfaces top channels, reactions, threads, DMs, inactive channels, and new team members. Open the Insights product from the team switcher to view them."), nil
	default:
		return ephemeralResponse("Unknown subcommand. Available: help"), nil
	}
}

func ephemeralResponse(text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}
