package consensus

import (
	"hxy352/src/log"
)

type CogsWorthPacemaker struct {
	consensus *BasicHotStuffWithCogsworth
}

func NewCogsWorthPacemaker(consensus *BasicHotStuffWithCogsworth) *CogsWorthPacemaker {
	c := &CogsWorthPacemaker{
		consensus: consensus,
	}

	return c
}

func (c *CogsWorthPacemaker) OnBeat() {
	for {
		select {
		case <-c.consensus.Ready:

			for {

				//c.consensus.ProcessCurrentViewQueue(c.consensus.CurrentView)

				req, ok := c.consensus.CmdCache.Dequeue()
				log.Infof("hs.CmdCache.Dequeue(): %+v", req)
				if !ok {
					continue
				}
				if c.consensus.CmdCache.IsFinished(req.GetCmd()) {
					log.Infof("hs.CmdCache.IsFinished(): %+v", req.GetCmd())
					continue
				}

				log.Infof("try to send prepare: %+v", req.GetCmd())
				c.consensus.SendPrepare(req)
				break
			}
		}
	}
}

func (c *CogsWorthPacemaker) WishToAdvance() {
	// todo: timeout * 2
	//c.timeout = service.NewTimeoutService(2 * c.timeout.Duration())
	c.consensus.timeout.Reset()
	c.consensus.timeout.Stop()

	// todo: if need create empty block???
	//hs.BlockChain.Store(hs.CreateLeaf(hs.CurrentBlock.Parent(), types.QuorumCert{}, ""))

	c.consensus.mut.Lock()

	c.consensus.CurrentView += 1

	c.consensus.CurrentBlock = nil

	// send wish next view msg
	c.consensus.SendWishNextView()

	c.consensus.mut.Unlock()

	c.consensus.timeout.SoftStart()
}
