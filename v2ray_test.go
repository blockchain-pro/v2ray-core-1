package core_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bitly/go-simplejson"
	proto "github.com/golang/protobuf/proto"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	. "v2ray.com/core"
	"v2ray.com/core/app/dispatcher"
	"v2ray.com/core/app/proxyman"
	"v2ray.com/core/common"
	"v2ray.com/core/common/net"
	"v2ray.com/core/common/protocol"
	"v2ray.com/core/common/serial"
	serial2 "v2ray.com/core/infra/conf/serial"

	"v2ray.com/core/common/uuid"
	"v2ray.com/core/features/dns"
	"v2ray.com/core/features/dns/localdns"
	_ "v2ray.com/core/main/distro/all"
	"v2ray.com/core/proxy/dokodemo"
	"v2ray.com/core/proxy/vmess"
	"v2ray.com/core/proxy/vmess/outbound"
	"v2ray.com/core/testing/servers/tcp"
)

func TestV2RayDependency(t *testing.T) {
	instance := new(Instance)

	wait := make(chan bool, 1)
	instance.RequireFeatures(func(d dns.Client) {
		if d == nil {
			t.Error("expected dns client fulfilled, but actually nil")
		}
		wait <- true
	})
	instance.AddFeature(localdns.New())
	<-wait
}

func TestV2RayClose(t *testing.T) {
	port := tcp.PickPort()

	userId := uuid.New()
	config := &Config{
		App: []*serial.TypedMessage{
			serial.ToTypedMessage(&dispatcher.Config{}),
			serial.ToTypedMessage(&proxyman.InboundConfig{}),
			serial.ToTypedMessage(&proxyman.OutboundConfig{}),
		},
		Inbound: []*InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortRange: net.SinglePortRange(port),
					Listen:    net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&dokodemo.Config{
					Address: net.NewIPOrDomain(net.LocalHostIP),
					Port:    uint32(0),
					NetworkList: &net.NetworkList{
						Network: []net.Network{net.Network_TCP, net.Network_UDP},
					},
				}),
			},
		},
		Outbound: []*OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&outbound.Config{
					Receiver: []*protocol.ServerEndpoint{
						{
							Address: net.NewIPOrDomain(net.LocalHostIP),
							Port:    uint32(0),
							User: []*protocol.User{
								{
									Account: serial.ToTypedMessage(&vmess.Account{
										Id: userId.String(),
									}),
								},
							},
						},
					},
				}),
			},
		},
	}

	cfgBytes, err := proto.Marshal(config)
	common.Must(err)

	server, err := StartInstance("protobuf", cfgBytes)
	common.Must(err)
	server.Close()
}

func TestV2RayRun(t *testing.T) {

	content, err := os.ReadFile("./v2Config.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	//log.Println(content)
	js, er1 := simplejson.NewJson(content)
	if er1 != nil {
		log.Fatal(er1)
	}

	vmess_list := js.Get("vmess").MustArray()

	for i, vmItem := range vmess_list {
		it, ok := vmItem.(map[string]interface{})
		if !ok {
			log.Fatal("type error!")
		}

		cfgjson, _ := json.Marshal(it)
		cfgReader := strings.NewReader(string(cfgjson))
		serverConfig1, er1 := serial2.LoadJSONConfig(cfgReader)
		if er1 != nil {
			log.Fatal(er1)
		}
		//serverConfig2 := &Config{
		//	App: []*serial.TypedMessage{
		//		serial.ToTypedMessage(&dispatcher.Config{}),
		//		serial.ToTypedMessage(&proxyman.InboundConfig{}),
		//		serial.ToTypedMessage(&proxyman.OutboundConfig{}),
		//	},
		//	Inbound: []*InboundHandlerConfig{
		//		{
		//			ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
		//				PortRange: net.SinglePortRange(port),
		//				Listen:    net.NewIPOrDomain(net.LocalHostIP),
		//			}),
		//
		//			ProxySettings: serial.ToTypedMessage(&dokodemo.Config{
		//				Address: net.NewIPOrDomain(serverAddr),
		//				Port:    uint32(port),
		//				NetworkList: &net.NetworkList{
		//					Network: []net.Network{net.Network_TCP},
		//				},
		//			}),
		//		},
		//	},
		//	Outbound: []*OutboundHandlerConfig{
		//		{
		//			ProxySettings: serial.ToTypedMessage(&outbound.Config{
		//
		//				Receiver: []*protocol.ServerEndpoint{
		//					{
		//						Address: net.NewIPOrDomain(serverAddr),
		//						Port:    uint32(serverPort),
		//						User: []*protocol.User{
		//							{
		//								Account: serial.ToTypedMessage(&vmess.Account{
		//									Id:      userID,
		//									AlterId: uint32(alterId),
		//									SecuritySettings: &protocol.SecurityConfig{
		//										Type: protocol.SecurityType_AUTO,
		//									},
		//								}),
		//							},
		//						},
		//					},
		//				},
		//			}),
		//
		//			SenderSettings: serial.ToTypedMessage(&proxyman.SenderConfig{
		//				StreamSettings: &internet.StreamConfig{
		//					Protocol: internet.TransportProtocol_WebSocket,
		//					TransportSettings: []*internet.TransportConfig{
		//						{
		//							Protocol: internet.TransportProtocol_WebSocket,
		//							Settings: serial.ToTypedMessage(&websocket.Config{}),
		//						},
		//					},
		//					//SecurityType: serial.GetMessageType(&tls.Config{}),
		//					//SecuritySettings: []*serial.TypedMessage{
		//					//	serial.ToTypedMessage(&tls.Config{
		//					//		AllowInsecure: false,
		//					//	}),
		//					//},
		//				},
		//			}),
		//		},
		//	},
		//}

		cfgBytes, err := proto.Marshal(serverConfig1)
		common.Must(err)

		server, err := StartInstance("protobuf", cfgBytes)
		if err != nil {
			log.Println(err)
			continue
		}
		//common.Must(err)
		//export http_proxy=http://127.0.0.1:1087;export https_proxy=http://127.0.0.1:1087;export ALL_PROXY=socks5://127.0.0.1:1080

		socket_proxy := fmt.Sprintf("127.0.0.1:%d", 10808)

		dialer, err := proxy.SOCKS5("tcp", socket_proxy, nil, proxy.Direct)
		dialContext := func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.Dial(network, address)
		}
		transport := &http.Transport{DialContext: dialContext,
			DisableKeepAlives: true}
		httpClient := &http.Client{Transport: transport}

		req, err := http.NewRequest("GET", "https://www.google.com/", nil)
		if err != nil {
			// handle error
			server.Close()
			log.Fatal(err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			log.Println(err)
			server.Close()
			continue
		}

		defer resp.Body.Close()
		defer server.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			// handle error
			server.Close()
			log.Fatal(err)
		}

		log.Println(i, string(body))

		server.Close()
	}

}
