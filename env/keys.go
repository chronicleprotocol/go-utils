package env

const (
	// LibP2P options
	KeyLibP2PPkSeed                 = "CFG_LIBP2P_PK_SEED"
	KeyLibP2PBootstrapAddrs         = "CFG_LIBP2P_BOOTSTRAP_ADDRS"
	KeyLibP2PRelayEnable            = "CFG_LIBP2P_RELAY_ENABLE"
	KeyLibP2PRelayServiceEnable     = "CFG_LIBP2P_RELAY_SERIVCE_ENABLE"
	KeyLibP2PNATServiceEnable       = "CFG_LIBP2P_NAT_SERIVCE_ENABLE"
	KeyLibP2PHolePunchingEnable     = "CFG_LIBP2P_HOLE_PUNCHING_ENABLE"
	KeyLibP2PDirectPeersAddrs       = "CFG_LIBP2P_DIRECT_PEERS_ADDRS"
	KeyLibP2PSubscriptionBufferSize = "CFG_LIBP2P_SUBSCRIPTION_BUFFER_SIZE"
	KeyLibP2PValidateQueueSize      = "CFG_LIBP2P_VALIDATE_QUEUE_SIZE"

	// Metrics options
	KeyMetricsAddr = "CFG_METRICS_ADDR"
	KeyMetricsHost = "CFG_METRICS_HOST"
	KeyMetricsPort = "CFG_METRICS_PORT"

	// Watch options
	KeyWatchInterval = "CFG_WATCH_INTERVAL"

	// General Env options
	KeyItemSeparator = "CFG_ITEM_SEPARATOR"

	// Rail options
	KeyRailMessageQueueSize = "CFG_RAIL_MESSAGE_QUEUE_SIZE"
	KeyRailEventQueueSize   = "CFG_RAIL_EVENT_QUEUE_SIZE"
)
