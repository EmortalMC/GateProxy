package command

import (
	"go.minekube.com/brigodier"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

func newShrugCmd(p *proxy.Proxy) brigodier.LiteralNodeBuilder {
	return brigodier.Literal("lobby").
		Executes(command.Command(func(c *command.Context) error {
			player, ok := c.Source.(proxy.Player)
			if !ok {
				c.Source.SendMessage(&Text{
					Content: "Shrug command cannot be used from console",
				})
				return nil
			}

			player.SpoofChatInput("¯\\_☻_/¯")

			return nil
		}))
}
