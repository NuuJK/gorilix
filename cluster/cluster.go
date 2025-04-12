package cluster

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
)



type SystemReference interface {
	SendMessage(ctx context.Context, actorID string, message interface{}) error
	SendNamedMessage(ctx context.Context, name string, message interface{}) error
}

type ClusterConfig struct {
	NodeName     string
	BindAddr     string
	BindPort     int
	Seeds        []string
	GossipNodes  int
	GossipPort   int
	PushInterval time.Duration
	PullInterval time.Duration
}

type Cluster struct {
	config     *ClusterConfig
	memberlist *memberlist.Memberlist
	events     chan memberlist.NodeEvent
	delegates  *clusterDelegate
	system     SystemReference
	nodesMutex sync.RWMutex
	nodes      map[string]*Node
	running    bool
}

type Node struct {
	Name    string
	Addr    net.IP
	Port    uint16
	Meta    map[string]string
	Status  NodeStatus
	ActorID string
}

type NodeStatus int

const (
	NodeAlive NodeStatus = iota
	NodeSuspect
	NodeDead
)


type clusterDelegate struct {
	broadcasts *memberlist.TransmitLimitedQueue
	msgCh      chan []byte
	metadata   map[string]string
	mtx        sync.RWMutex
}

func newClusterDelegate() *clusterDelegate {
	d := &clusterDelegate{
		msgCh:    make(chan []byte, 1024),
		metadata: make(map[string]string),
	}
	d.broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes:       func() int { return 3 },
		RetransmitMult: 3,
	}
	return d
}


func (d *clusterDelegate) NodeMeta(limit int) []byte {
	d.mtx.RLock()
	defer d.mtx.RUnlock()

	
	meta := make([]byte, 0, 128)
	for k, v := range d.metadata {
		if len(meta)+len(k)+len(v)+2 <= limit {
			meta = append(meta, byte(len(k)))
			meta = append(meta, k...)
			meta = append(meta, byte(len(v)))
			meta = append(meta, v...)
		}
	}
	return meta
}


func (d *clusterDelegate) NotifyMsg(msg []byte) {
	if len(msg) == 0 {
		return
	}

	select {
	case d.msgCh <- msg:
	default:
		
	}
}


func (d *clusterDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}


func (d *clusterDelegate) LocalState(join bool) []byte {
	return []byte{} 
}


func (d *clusterDelegate) MergeRemoteState(buf []byte, join bool) {
	
}

func DefaultConfig() *ClusterConfig {
	return &ClusterConfig{
		NodeName:     "",
		BindAddr:     "0.0.0.0",
		BindPort:     7946,
		GossipNodes:  3,
		GossipPort:   7946,
		PushInterval: 1 * time.Second,
		PullInterval: 3 * time.Second,
	}
}

func NewCluster(config *ClusterConfig, system SystemReference) *Cluster {
	if config == nil {
		config = DefaultConfig()
	}

	if config.NodeName == "" {
		hostname, err := os.Hostname()
		if err == nil {
			config.NodeName = hostname
		} else {
			config.NodeName = fmt.Sprintf("node-%d", time.Now().UnixNano())
		}
	}

	return &Cluster{
		config:    config,
		events:    make(chan memberlist.NodeEvent, 100),
		delegates: newClusterDelegate(),
		system:    system,
		nodes:     make(map[string]*Node),
		running:   false,
	}
}

func (c *Cluster) Start() error {
	if c.running {
		return fmt.Errorf("cluster already running")
	}

	conf := memberlist.DefaultLANConfig()
	conf.Name = c.config.NodeName
	conf.BindAddr = c.config.BindAddr
	conf.BindPort = c.config.BindPort
	conf.Events = &memberlist.ChannelEventDelegate{Ch: c.events}
	conf.Delegate = c.delegates
	conf.GossipNodes = c.config.GossipNodes
	conf.PushPullInterval = c.config.PullInterval
	conf.GossipInterval = c.config.PushInterval

	list, err := memberlist.Create(conf)
	if err != nil {
		return fmt.Errorf("failed to create memberlist: %w", err)
	}

	c.memberlist = list
	c.running = true

	if len(c.config.Seeds) > 0 {
		_, err = c.memberlist.Join(c.config.Seeds)
		if err != nil {
			return fmt.Errorf("failed to join cluster: %w", err)
		}
	}

	go c.handleEvents()

	return nil
}

func (c *Cluster) Stop() error {
	if !c.running {
		return nil
	}

	err := c.memberlist.Leave(time.Second * 5)
	if err != nil {
		return fmt.Errorf("error leaving cluster: %w", err)
	}

	err = c.memberlist.Shutdown()
	if err != nil {
		return fmt.Errorf("error shutting down memberlist: %w", err)
	}

	c.running = false
	close(c.events)
	return nil
}

func (c *Cluster) handleEvents() {
	for event := range c.events {
		c.nodesMutex.Lock()
		switch event.Event {
		case memberlist.NodeJoin:
			c.nodes[event.Node.Name] = &Node{
				Name:   event.Node.Name,
				Addr:   event.Node.Addr,
				Port:   event.Node.Port,
				Meta:   map[string]string{},
				Status: NodeAlive,
			}
		case memberlist.NodeLeave:
			if node, exists := c.nodes[event.Node.Name]; exists {
				node.Status = NodeSuspect
			}
		case memberlist.NodeUpdate:
			if node, exists := c.nodes[event.Node.Name]; exists {
				node.Addr = event.Node.Addr
				node.Port = event.Node.Port
			}
		default:
			
			if node, exists := c.nodes[event.Node.Name]; exists {
				node.Status = NodeSuspect
			}
		}
		c.nodesMutex.Unlock()
	}
}

func (c *Cluster) Members() []*Node {
	c.nodesMutex.RLock()
	defer c.nodesMutex.RUnlock()

	members := make([]*Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		members = append(members, node)
	}
	return members
}

func (c *Cluster) Join(seeds []string) (int, error) {
	if !c.running {
		return 0, fmt.Errorf("cluster not running")
	}
	return c.memberlist.Join(seeds)
}

func (c *Cluster) Leave(timeout time.Duration) error {
	if !c.running {
		return nil
	}
	return c.memberlist.Leave(timeout)
}

func (c *Cluster) Self() *Node {
	self := c.memberlist.LocalNode()
	return &Node{
		Name:   self.Name,
		Addr:   self.Addr,
		Port:   self.Port,
		Status: NodeAlive,
	}
}

func (c *Cluster) GetNode(name string) (*Node, bool) {
	c.nodesMutex.RLock()
	defer c.nodesMutex.RUnlock()

	node, exists := c.nodes[name]
	return node, exists
}

func (c *Cluster) BroadcastMessage(msg []byte) error {
	if !c.running {
		return fmt.Errorf("cluster not running")
	}

	c.delegates.broadcasts.QueueBroadcast(&broadcast{
		msg:    msg,
		notify: nil,
	})
	return nil
}


type broadcast struct {
	msg    []byte
	notify chan<- struct{}
}

func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
	return false
}

func (b *broadcast) Message() []byte {
	return b.msg
}

func (b *broadcast) Finished() {
	if b.notify != nil {
		close(b.notify)
	}
}

func (c *Cluster) SendToNode(nodeName string, msg []byte) error {
	if !c.running {
		return fmt.Errorf("cluster not running")
	}

	c.nodesMutex.RLock()
	defer c.nodesMutex.RUnlock()

	for _, node := range c.memberlist.Members() {
		if node.Name == nodeName {
			return c.memberlist.SendReliable(node, msg)
		}
	}

	return fmt.Errorf("node %s not found", nodeName)
}
