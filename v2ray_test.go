package core_test

import (
	"bytes"
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
	"reflect"
	"testing"
	. "v2ray.com/core"
	"v2ray.com/core/app/dispatcher"
	"v2ray.com/core/app/proxyman"
	"v2ray.com/core/common"
	"v2ray.com/core/common/net"
	"v2ray.com/core/common/protocol"
	"v2ray.com/core/common/serial"
	"v2ray.com/core/common/uuid"
	"v2ray.com/core/features/dns"
	"v2ray.com/core/features/dns/localdns"
	_ "v2ray.com/core/main/distro/all"
	"v2ray.com/core/proxy/dokodemo"
	"v2ray.com/core/proxy/vmess"
	"v2ray.com/core/proxy/vmess/outbound"
	"v2ray.com/core/testing/servers/tcp"
	"v2ray.com/core/transport/internet"
	"v2ray.com/core/transport/internet/tls"
	"v2ray.com/core/transport/internet/websocket"
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

	content, err := os.ReadFile("./guiNConfig.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	//log.Println(content)
	js, er1 := simplejson.NewJson(content)
	if er1 != nil {
		log.Fatal(er1)
	}

	inbounds := js.Get("inbound").MustArray()

	var localPort int64 = 0
	if len(inbounds) > 0 {
		log.Println(reflect.TypeOf(inbounds[0].(map[string]interface{})["localPort"]))
		localPort, _ = inbounds[0].(map[string]interface{})["localPort"].(json.Number).Int64()
	}

	port, _ := net.PortFromInt(uint32(localPort)) //tcp.PickPort()
	//port := tcp.PickPort()

	vmess_list := js.Get("vmess").MustArray()
	for i, vmItem := range vmess_list {
		it, ok := vmItem.(map[string]interface{})
		if !ok {
			log.Fatal("type error!")
		}
		itdata, _ := json.Marshal(it)
		vmJson, _ := simplejson.NewJson(itdata)
		addr := vmJson.Get("address").MustString()
		serverAddr := net.ParseAddress(addr)
		serverPort := vmJson.Get("port").MustInt()
		userID := vmJson.Get("id").MustString()
		alterId := vmJson.Get("alterId").MustInt()

		//serverConfig1 := &Config{
		//	App: []*serial.TypedMessage{
		//		serial.ToTypedMessage(&v2log.Config{
		//			ErrorLogLevel: clog.Severity_Debug,
		//			ErrorLogType:  v2log.LogType_Console,
		//		}),
		//	},
		//	Inbound: []*InboundHandlerConfig{
		//		{
		//			ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
		//				PortRange: net.SinglePortRange(port),
		//				Listen:    net.NewIPOrDomain(net.LocalHostIP),
		//				StreamSettings: &internet.StreamConfig{
		//					Protocol: internet.TransportProtocol_WebSocket,
		//					TransportSettings: []*internet.TransportConfig{
		//						{
		//							Protocol: internet.TransportProtocol_WebSocket,
		//						},
		//					},
		//				},
		//			}),
		//			ProxySettings: serial.ToTypedMessage(&inbound.Config{
		//				User: []*protocol.User{
		//					{
		//						Account: serial.ToTypedMessage(&vmess.Account{
		//							Id:      userID,
		//							AlterId: uint32(alterId),
		//							SecuritySettings: &protocol.SecurityConfig{
		//								Type: protocol.SecurityType_AUTO,
		//							},
		//						}),
		//					},
		//				},
		//			}),
		//		},
		//	},
		//	Outbound: []*OutboundHandlerConfig{
		//		{
		//			ProxySettings: serial.ToTypedMessage(&freedom.Config{}),
		//		},
		//	},
		//}

		serverConfig2 := &Config{
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
						StreamSettings: &internet.StreamConfig{
							Protocol: internet.TransportProtocol_WebSocket,
							TransportSettings: []*internet.TransportConfig{
								{
									Protocol: internet.TransportProtocol_WebSocket,
									Settings: serial.ToTypedMessage(&websocket.Config{}),
								},
							},
							SecurityType: serial.GetMessageType(&tls.Config{}),
							SecuritySettings: []*serial.TypedMessage{
								serial.ToTypedMessage(&tls.Config{
									AllowInsecure: true,
								}),
							},
						},
					}),

					ProxySettings: serial.ToTypedMessage(&dokodemo.Config{
						Address: net.NewIPOrDomain(serverAddr),
						Port:    uint32(port),
						NetworkList: &net.NetworkList{
							Network: []net.Network{net.Network_TCP},
						},
					}),
				},
			},
			Outbound: []*OutboundHandlerConfig{
				{
					ProxySettings: serial.ToTypedMessage(&outbound.Config{

						Receiver: []*protocol.ServerEndpoint{
							{
								Address: net.NewIPOrDomain(serverAddr),
								Port:    uint32(serverPort),
								User: []*protocol.User{
									{
										Account: serial.ToTypedMessage(&vmess.Account{
											Id:      userID,
											AlterId: uint32(alterId),
											SecuritySettings: &protocol.SecurityConfig{
												Type: protocol.SecurityType_AUTO,
											},
										}),
									},
								},
							},
						},
					}),

					SenderSettings: serial.ToTypedMessage(&proxyman.SenderConfig{
						StreamSettings: &internet.StreamConfig{
							Protocol: internet.TransportProtocol_WebSocket,
							TransportSettings: []*internet.TransportConfig{
								{
									Protocol: internet.TransportProtocol_WebSocket,
									Settings: serial.ToTypedMessage(&websocket.Config{}),
								},
							},
							//SecurityType: serial.GetMessageType(&tls.Config{}),
							//SecuritySettings: []*serial.TypedMessage{
							//	serial.ToTypedMessage(&tls.Config{
							//		AllowInsecure: false,
							//	}),
							//},
						},
					}),
				},
			},
		}

		cfgBytes, err := proto.Marshal(serverConfig2)
		common.Must(err)

		server, err := StartInstance("protobuf", cfgBytes)
		if err != nil {
			log.Println(err)
			continue
		}
		//common.Must(err)
		//export http_proxy=http://127.0.0.1:1087;export https_proxy=http://127.0.0.1:1087;export ALL_PROXY=socks5://127.0.0.1:1080
		socket_proxy := fmt.Sprintf("127.0.0.1:%d", localPort)

		dialer, err := proxy.SOCKS5("tcp", socket_proxy, nil, proxy.Direct)
		dialContext := func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.Dial(network, address)
		}
		transport := &http.Transport{DialContext: dialContext,
			DisableKeepAlives: true}
		httpClient := &http.Client{Transport: transport}

		//POST /api/gimme/ HTTP/1.1
		//Accept: */*
		//Accept-Encoding: gzip, deflate, br
		//Accept-Language: zh-CN,zh;q=0.9,en;q=0.8
		//Connection: keep-alive
		//Content-Length: 615
		//Content-Type: application/json
		//Host: faucet.egorfine.com
		//Origin: https://faucet.egorfine.com
		//Referer: https://faucet.egorfine.com/
		//Sec-Fetch-Dest: empty
		//Sec-Fetch-Mode: cors
		//Sec-Fetch-Site: same-origin
		//User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36
		//sec-ch-ua: "Chromium";v="104", " Not A;Brand";v="99", "Google Chrome";v="104"
		//sec-ch-ua-mobile: ?0
		//sec-ch-ua-platform: "macOS"

		//{"address":"0x262768022cC7F141C91C9a6970c55b22aa6e473C","token":"03AIIukzjuE8YX-P-JtqmmL8B_GxwbzkWYQsVmCpLvVME0sjCYPk6bb_wI4KTJKuZRXScrRvPpUNL5-7172RvRsftNNFR-qs0RajQzZn0RcupkBVkAeHq0MnJZLpZsCUG0E79pQx4O0ahYdkcT0VmHigYsuLCliLHbZDsIbwdCcvltIQBJxdvloI6KVQSZRP5uBQtmRacB9vjdkriG_gFqH4XXoCgiWX3d_FS7h-pwbXqZ2A_uKE89LEN5LTAHKxNVtQe1PrPlEvp0AwZRt_yupVTUc202emmplMQMdlqjWBzri5dyUgfC82wQglI42NSbDiuDhh-2S3FgBe5zIxhHJW7P70ZkXCD_AebfsJP2fcmOdrnBCtg3zidNF_vBV5Yc8C2VPT7EadNkq17_IfkCuIr3TQZBIFuAMPysHbHZQUqLy7zBg9A63GnpHry3oxba6HK96XrAO1LXsvRLiVewJ2fh-lOHD-cfRCWkTx52AOb0HL1kxk_Vg9lysQFY9puUg8g4Y5o9DZrXb1ontPZnDDDdP8x6mplZug"}
		var bodyJson map[string]interface{} = map[string]interface{}{"address": "0x5528d87AD870b9D06D616F788DfFa02c40bd9466",
			"token": "03AIIukzgLvgAgeL4alaR1i-dwP8AI7x6DI8oLyJ76aLilStwH6zNmrSCSyalHqANAVRmwIOTQJJfEZzDeB-xXFtqlAoMbIKU1wSIJsA_0GrlSETk1UwRs5s1XfARu6sWMw13D6Su0RihCOQW2hjCUH5LwtcGplnpoV3gVosGugl7K_tf-g6EDr4GApTKjK-5mag7HFahk8ViXNrlB_QOAQGfjTD0TdQ7O7uzHneOJsrYmrOVtN2OBpA1p9rCiS3VTlsrXodBhxbMu5hethw34SkRwpuxxHbhRFzC4Z4Jyy1YP3uGmHN6j867aoHBOhCbrgPoFg3mjDmO46QOm1lvmz30Db6ulK7beJJgqL316CN2rbx9Dgd3kh-GQ5NKnmzNHqdbN_hRPJxxfqJ4LVY7WpF6zx9KZepp6W0F5ZrwEaQ6MaHcpOg9pwqsrvbWstAeCBoKVPiIGPL5NIPv-h9bR3z13zo6d7II7cCHE3lst9ZQ0JRTdYEGkARNDIx7Q6xaWnwAuUgUbwawDybX_s7xF_QlYSVRdc3oONgJEGzOgGFSI4wA-VrPjdu9kkbVfFXQGkoIbKxuyWkUCzU53iDGMVxEthfZFTJ_STcnPamcdigjApEKDh1mwYg8OX4JrA9Sf1o2pDlf696svya_3MYD_xqf9-Pj98kwTbRPxzxKRGEY-hYTEVs8-TfFJZG89WWiqxrsv2RwTDuq4CJ-2cAV5pCkaM_yMZQEPnzOZU-rHMcZw_Mq9EIOHAalt1tvaBP4CX-kVDEpeqcrx0uEv8G3l_D0x_UGl8_I7k-SE3vmM2kNWHenDsxl1-e9j7QM-suOGz4aQ0Q049HLn0DDsR2JwzfBxTzNJe_BpNLTYWqfCmltXgqiv34_BeYC1LdLBtBbZ6E4X0Y4qMC6jm3M0tFikjwaFWAnX8h_eR6s5j9uPkn1Y0Kbnd2Q8fSCs8u0JYbi9OWawGgjdONI7gKoTYKMLk6UT3gyVWIOn2l8TH6TRExarYGocfUKss2zFuqpJfu8YveelJteQGA5s0eX4l9fgTtPG7snY6s0jIWcPBEg6v4L0wn-wEogQd18L3NNMXLmJAUwHwVXWyZycsnMgtn7ToC2_Kcveo14THy2nvb5svNfexHTqkgvCZ6mCO4oEBDSgpduvpe03B9RL29qZmViYnWkEqaoIN1Vj1Zuepl7nPu6LZLAT1crJJ-AjLFrd1SrBzSaDJPp1O-53wSSm_zOI8Cb_WuveFfkJzZPGfscmDu3z0d7DHZtdM5YDfZLP3j4wKtlkLrnKxqJROi3WMCz8SVZAvmu3CR8qkilYfyZnUD6pT7WRrj69wf5FeeS3k3CRBp-W9uiAS5GWmiLOzyQPsJbCsTBs1CpaFXELeahGoFXJRzl-TSIthJ1g0BI0-t9cPUyXVbuROOhA3fLjyWnPI9zV4KxQdsQunDagpElYaNSzqliWr1x7kMaXRhgElBXiN1loZpiP7zcBRwNzvbqOKZDeShVgbpxx63tAAe5ETblSdRcfs_WR5rs",
		}

		bodyData, _ := json.Marshal(bodyJson)

		bodyReader := bytes.NewBuffer(bodyData)

		req, err := http.NewRequest("POST", "https://faucet.egorfine.com/api/gimme/", bodyReader)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("User-Agent", "PostmanRuntime/7.29.2")

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
