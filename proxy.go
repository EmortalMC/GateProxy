package main

import (
	"context"
	"fmt"
	"go.minekube.com/gate/pkg/edition/java/ping"
	"go.minekube.com/gate/pkg/edition/java/proto/state"
	"go.minekube.com/gate/pkg/edition/java/proto/version"
	"math"
	"math/rand"
	"simple-proxy/command"
	"simple-proxy/game"
	"simple-proxy/redisdb"
	"simple-proxy/webhook"
	"time"

	"github.com/robinbraemer/event"
	"go.minekube.com/common/minecraft/color"
	. "go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/cmd/gate"
	"go.minekube.com/gate/pkg/edition/java/proxy"
)

var webhook_url = ""

func main() {

	redisdb.RedisClient = redisdb.InitRedis()

	state.Play.ClientBound.Register(&command.EntitySoundEffect{}, &state.PacketMapping{
		ID:       0x5F,
		Protocol: version.Minecraft_1_19_1.Protocol,
	})

	proxy.Plugins = append(proxy.Plugins, proxy.Plugin{
		Name: "SimpleProxy",
		Init: func(ctx context.Context, proxy *proxy.Proxy) error {
			return newSimpleProxy(proxy).init()
		},
	})

	// Execute Gate entrypoint and block until shutdown.
	// We could also run gate.Start if we don't need Gate's command-line.
	gate.Execute()
}

type SimpleProxy struct {
	*proxy.Proxy
}

func newSimpleProxy(proxy *proxy.Proxy) *SimpleProxy {
	return &SimpleProxy{
		Proxy: proxy,
	}
}

func (p *SimpleProxy) init() error {
	command.RegisterCommands(p.Proxy)
	p.registerSubscribers()

	game.RegisterPubSub(p.Proxy)
	return nil
}

// Register event subscribers
func (p *SimpleProxy) registerSubscribers() {
	// Send message on server switch.
	event.Subscribe(p.Event(), 0, p.onServerLogin)
	event.Subscribe(p.Event(), 0, p.onServerDisconnect)
	event.Subscribe(p.Event(), 0, p.onChat)

	// Change the MOTD response.
	event.Subscribe(p.Event(), 0, pingHandler(p.Proxy))
}

func (p *SimpleProxy) onServerLogin(e *proxy.ServerConnectedEvent) {
	refreshTablist(p.Proxy)
	webhook.PlayerJoined(e.Player(), p.PlayerCount(), webhook_url)
}

func (p *SimpleProxy) onServerDisconnect(e *proxy.DisconnectEvent) {
	refreshTablist(p.Proxy)
	webhook.PlayerLeft(e.Player(), p.PlayerCount(), webhook_url)
}

func (p *SimpleProxy) onChat(e *proxy.PlayerChatEvent) {
	e.SetAllowed(false)
	for _, plr := range p.Players() {
		go plr.SendMessage(&Text{
			Extra: []Component{
				&Text{
					S:       Style{Color: color.Gray},
					Content: e.Player().Username(),
				},
				&Text{
					S:       Style{Color: color.Gray},
					Content: ": ",
				},
				&Text{
					S:       Style{Color: color.White},
					Content: e.Message(),
				},
			},
		})
	}
}

func refreshTablist(p *proxy.Proxy) {
	purple, _ := color.Make(color.LightPurple)
	gold, _ := color.Make(color.Gold)
	aqua, _ := color.Make(color.Aqua)
	ip, _ := color.Hex("#266ee0")

	for _, plr := range p.Players() {
		plr.TabList().SetHeaderFooter(
			&Text{
				Extra: []Component{
					&Text{
						S:       Style{Color: gold},
						Content: "┌                                                  ",
					},
					&Text{
						S:       Style{Color: purple},
						Content: "┐ \n",
					},
					command.Gradient("EmortalMC\n", Style{Bold: True}, *gold, *purple, *aqua),
				},
			},
			&Text{
				Extra: []Component{
					&Text{
						S:       Style{Color: color.Gray},
						Content: fmt.Sprintf(" \n%d online", p.PlayerCount()),
					},
					&Text{
						S:       Style{Color: ip},
						Content: "\nmc.emortal.dev",
					},
					&Text{
						S:       Style{Color: purple},
						Content: "\n└                                                  ",
					},
					&Text{
						S:       Style{Color: gold},
						Content: "┘ ",
					},
				},
			},
		)
	}
}

func pingHandler(p *proxy.Proxy) func(evt *proxy.PingEvent) {
	messages := []string{
		"coolest server to ever exist",
		"better than hypixel",
		"you should join",
		"stop scrolling, click here!",
		"Lunar client users: Beware!",
		"using 3 server softwares!",
		"gradient lover",
		"emortal is watching",
		"Chuck Norris joined and said it was pretty good",
	}

	return func(e *proxy.PingEvent) {
		randomMessage := messages[rand.Intn(len(messages))]

		first, _ := color.Make(color.Gold)
		second, _ := color.Make(color.LightPurple)
		third, _ := color.Make(color.Aqua)

		motd := &Text{
			Extra: []Component{
				&Text{
					Content: "▓▒░              ",
					S:       Style{Color: color.LightPurple},
				},
				&Text{
					Content: "⚡   ",
					S:       Style{Color: color.LightPurple, Bold: True},
				},
				command.Gradient("EmortalMC", Style{Bold: True}, *first, *second, *third),
				&Text{
					Content: "   ⚡",
					S:       Style{Color: color.Gold, Bold: True},
				},
				&Text{
					Content: "              ░▒▓",
					S:       Style{Color: color.Gold},
				},
				&Text{
					Content: "\n" + randomMessage,
					S:       Style{Color: color.Yellow},
				},
			},
		}

		evt := e.Ping()
		evt.Description = motd
		evt.Players.Max = evt.Players.Online + 1

		sampleCount := int(math.Min(float64(p.PlayerCount()), 10))
		samples := make([]ping.SamplePlayer, sampleCount)
		for i, plr := range p.Players() {
			if i >= sampleCount {
				break
			}
			samples[i] = ping.SamplePlayer{Name: plr.Username(), ID: plr.ID()}
		}

		evt.Players.Sample = samples
	}
}

// tick runs a function every interval until the context is cancelled.
func tick(ctx context.Context, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			fn()
		case <-ctx.Done():
			return
		}
	}
}

func tickB(ctx context.Context, ticks int, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	i := 0
	for i < ticks {
		select {
		case <-ticker.C:
			i++
			fn()
		case <-ctx.Done():
			return
		}
	}
}
