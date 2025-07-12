package service

//import (
//	"hxy352/src/proto/hotstuffpb"
//	"hxy352/src/types"
//)

//
//// HotStuffReplica Replica provides methods used by hotstuff to send messages to replicas.
//type HotStuffReplica struct {
//	eventLoop *eventloop.EventLoop
//	node      *hotstuffpb.Node
//	id        types.ID
//	pubKey    types.PublicKey
//	md        map[string]string
//}
//
//// ID returns the replica's ID.
//func (r *HotStuffReplica) ID() types.ID {
//	return r.id
//}
//
//// PublicKey returns the replica's public key.
//func (r *HotStuffReplica) PublicKey() types.PublicKey {
//	return r.pubKey
//}
//
//// Vote sends the partial certificate to the other replica.
//func (r *HotStuffReplica) Vote(cert types.PartialCert) {
//	if r.node == nil {
//		return
//	}
//	ctx, cancel := synchronizer.TimeoutContext(r.eventLoop.Context(), r.eventLoop)
//	defer cancel()
//	pCert := hotstuffpb.PartialCertToProto(cert)
//	r.node.Vote(ctx, pCert)
//}
//
//// NewView sends the quorum certificate to the other replica.
//func (r *HotStuffReplica) NewView(msg types.SyncInfo) {
//	if r.node == nil {
//		return
//	}
//	ctx, cancel := synchronizer.TimeoutContext(r.eventLoop.Context(), r.eventLoop)
//	defer cancel()
//	r.node.NewView(ctx, hotstuffpb.SyncInfoToProto(msg))
//}
//
//// Metadata returns the gRPC metadata from this replica's connection.
//func (r *HotStuffReplica) Metadata() map[string]string {
//	return r.md
//}
