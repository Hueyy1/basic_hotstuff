package service

import (
	"hxy352/src/log"
	"hxy352/src/proto/basichotstuffpb"
)

type CmdCache struct {
	finished map[string]bool // todo: expire key
	//queue    *Queue[*basichotstuffpb.Request]
	channel chan *basichotstuffpb.Request
}

func NewCmdCache() *CmdCache {
	return &CmdCache{
		finished: make(map[string]bool),
		//queue:    NewQueue[*basichotstuffpb.Request](),
		channel: make(chan *basichotstuffpb.Request, 500),
	}
}

func (c *CmdCache) Enqueue(req *basichotstuffpb.Request) {
	//c.queue.Enqueue(req)
	log.Debugf("Enqueue req: %+v", req)
	c.channel <- req
}

func (c *CmdCache) Dequeue() (req *basichotstuffpb.Request, ok bool) {
	//return c.queue.Dequeue()
	return <-c.channel, true
}

func (c *CmdCache) Finish(cmd string) {
	c.finished[cmd] = true
}

func (c *CmdCache) IsFinished(cmd string) bool {
	return c.finished[cmd]
}
