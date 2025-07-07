package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/relab/gorums"
	"github.com/spf13/cobra"
	"hxy352/src/crypto/bls12"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/server"
	"hxy352/src/types"
	"net"
	"strconv"
)

var hsAnnotations = map[string]string{"app": "hotstuff", "isGraceful": "true"}

func newBasicHotStuffServiceCmd() *cobra.Command {
	//var useTls bool
	cmd := &cobra.Command{
		Use:         "bhs",
		Long:        "basic hotstuff",
		RunE:        startBasicHotStuffService,
		Annotations: hsAnnotations,
	}
	//cmd.Flags().BoolVarP(&useTls, "use-tls", "u", true, "use tls")
	return cmd
}

func init() {
	rootCmd.AddCommand(newBasicHotStuffServiceCmd())
}

func startBasicHotStuffService(cmd *cobra.Command, _ []string) (err error) {
	gCfg := LoadConfig()
	log.Infof("config is %+v", gCfg)

	id, _ := cmd.Flags().GetInt("id")
	replicaConf := &model.ReplicaConf{
		Id:     types.ID(id),
		Conf:   gCfg.Replica[id],
		UseTLS: false,
	}

	replicaConf.CaKey, replicaConf.Ca, err = bls12.GenerateCA()
	if err != nil {
		return err
	}

	err = SetReplicaCertificates(replicaConf)
	if err != nil {
		return err
	}

	err = createReplica(replicaConf)
	if err != nil {
		return err
	}

	log.Infof("Created replica at port %d", replicaConf.Conf.Port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", replicaConf.Conf.Port))
	if err != nil {
		log.Panic(err)
	}
	gorumsSrv := gorums.NewServer()
	srv := server.NewBasicHotStuffImpl(replicaConf, &gCfg)
	basichotstuffpb.RegisterBasicHotStuffServer(gorumsSrv, srv)
	gorumsSrv.Serve(lis)

	//phoneValidationService := service.NewPhoneValidationService()
	//server.NewPhoneValidationServiceServer(&gCfg, phoneValidationService, useTls)

	return nil
}

func SetReplicaCertificates(replicaConf *model.ReplicaConf) error {
	replicaConf.CertificateAuthority = bls12.CertToPEM(replicaConf.Ca)
	validFor := []string{"localhost", "127.0.0.1", "0.0.0.0"}

	replicaConf.Crypto = "bls12" // default crypto

	// todo: bug
	keyChain, err := bls12.GenerateKeyChain(types.ID(replicaConf.Conf.Id), validFor, replicaConf.Crypto, replicaConf.Ca, replicaConf.CaKey)
	if err != nil {
		return fmt.Errorf("failed to generate keychain: %w", err)
	}

	replicaConf.PrivateKey = keyChain.PrivateKey
	replicaConf.PublicKey = keyChain.PublicKey
	replicaConf.Certificate = keyChain.Certificate
	replicaConf.CertificateKey = keyChain.CertificateKey
	return nil
}

func createReplica(replicaConf *model.ReplicaConf) error {

	// get private key and certificates
	privKey, err := bls12.ParsePrivateKey(replicaConf.PrivateKey)
	if err != nil {
		return err
	}
	var certificate tls.Certificate
	var rootCAs *x509.CertPool
	log.Debugf("Using TLS: %v", replicaConf.UseTLS)
	fmt.Println(string(replicaConf.CertificateKey))
	if replicaConf.UseTLS {
		certificate, err = tls.X509KeyPair(replicaConf.Certificate, replicaConf.CertificateKey)
		if err != nil {
			return err
		}
		rootCAs = x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(replicaConf.CertificateAuthority)
	}
	replicaConf.RootCAs = rootCAs
	replicaConf.PrivKey = privKey
	replicaConf.TlsCertificate = certificate

	return nil

	//// prepare modules
	//builder := modules.NewBuilder(hotstuff.ID(opts.GetID()), privKey)
	//
	//consensusRules, ok := modules.GetModule[consensus.Rules](opts.GetConsensus())
	//if !ok {
	//	return nil, fmt.Errorf("invalid consensus name: '%s'", opts.GetConsensus())
	//}
	//
	//strategy := opts.GetByzantineStrategy()
	//if strategy != "" {
	//	if byz, ok := modules.GetModule[byzantine.Byzantine](strategy); ok {
	//		consensusRules = byz.Wrap(consensusRules)
	//		logger.Infof("assigned byzantine strategy: %s", strategy)
	//
	//	} else {
	//		return nil, fmt.Errorf("invalid byzantine strategy: '%s'", opts.GetByzantineStrategy())
	//	}
	//}
	//
	//cryptoImpl, ok := modules.GetModule[modules.CryptoBase](opts.GetCrypto())
	//if !ok {
	//	return nil, fmt.Errorf("invalid crypto name: '%s'", opts.GetCrypto())
	//}
	//
	//leaderRotation, ok := modules.GetModule[modules.LeaderRotation](opts.GetLeaderRotation())
	//if !ok {
	//	return nil, fmt.Errorf("invalid leader-rotation algorithm: '%s'", opts.GetLeaderRotation())
	//}
	//var viewDuration synchronizer.ViewDuration
	//if opts.GetLeaderRotation() == "tree-leader" {
	//	// TODO(meling): Temporary default; should be configurable and moved to the appropriate place.
	//	opts.SetTreeHeightWaitTime()
	//	// create tree only if we are using tree leader (Kauri)
	//	builder.Options().SetTree(createTree(opts))
	//	viewDuration = synchronizer.NewFixedViewDuration(opts.GetInitialTimeout().AsDuration())
	//} else {
	//	viewDuration = synchronizer.NewViewDuration(
	//		uint64(opts.GetTimeoutSamples()),
	//		float64(opts.GetInitialTimeout().AsDuration().Nanoseconds())/float64(time.Millisecond),
	//		float64(opts.GetMaxTimeout().AsDuration().Nanoseconds())/float64(time.Millisecond),
	//		float64(opts.GetTimeoutMultiplier()),
	//	)
	//}
	//sync := synchronizer.New(viewDuration)
	//builder.Add(
	//	eventloop.New(1000),
	//	consensus.New(consensusRules),
	//	consensus.NewVotingMachine(),
	//	crypto.NewCache(cryptoImpl, 100), // TODO: consider making this configurable
	//	leaderRotation,
	//	sync,
	//	w.metricsLogger,
	//	blockchain.New(),
	//	logger,
	//)
	//builder.Options().SetSharedRandomSeed(opts.GetSharedSeed())
	//
	//if w.measurementInterval > 0 {
	//	replicaMetrics := metrics.GetReplicaMetrics(w.metrics...)
	//	builder.Add(replicaMetrics...)
	//	builder.Add(metrics.NewTicker(w.measurementInterval))
	//}
	//
	//for _, n := range opts.GetModules() {
	//	m, ok := modules.GetModuleUntyped(n)
	//	if !ok {
	//		return nil, fmt.Errorf("no module named '%s'", n)
	//	}
	//	builder.Add(m)
	//}
	//c := replica.Config{
	//	ID:          hotstuff.ID(opts.GetID()),
	//	PrivateKey:  privKey,
	//	TLS:         opts.GetUseTLS(),
	//	Certificate: &certificate,
	//	RootCAs:     rootCAs,
	//	Locations:   opts.GetLocations(),
	//	BatchSize:   opts.GetBatchSize(),
	//	ManagerOptions: []gorums.ManagerOption{
	//		gorums.WithDialTimeout(opts.GetConnectTimeout().AsDuration()),
	//	},
	//}
	//return replica.New(c, builder), nil
}

func getPort(lis net.Listener) (uint32, error) {
	_, portStr, err := net.SplitHostPort(lis.Addr().String())
	if err != nil {
		return 0, err
	}
	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(port), nil
}
