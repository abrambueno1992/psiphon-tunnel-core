/*
 * Copyright (c) 2015, Psiphon Inc.
 * All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package psiphon

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon/common"
	"github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon/common/parameters"
	"github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon/common/protocol"
)

const (
	TUNNEL_POOL_SIZE = 1
)

// Config is the Psiphon configuration specified by the application. This
// configuration controls the behavior of the core tunnel functionality.
//
// To distinguish omitted timeout params from explicit 0 value timeout params,
// corresponding fieldss are int pointers. nil means no value was supplied and
// to use the default; a non-nil pointer to 0 means no timeout.
type Config struct {

	// DataStoreDirectory is the directory in which to store the persistent
	// database, which contains information such as server entries. By
	// default, current working directory.
	//
	// Warning: If the datastore file, DataStoreDirectory/DATA_STORE_FILENAME,
	// exists but fails to open for any reason (checksum error, unexpected
	// file format, etc.) it will be deleted in order to pave a new datastore
	// and continue running.
	DataStoreDirectory string

	// PropagationChannelId is a string identifier which indicates how the
	// Psiphon client was distributed. This parameter is required. This value
	// is supplied by and depends on the Psiphon Network, and is typically
	// embedded in the client binary.
	PropagationChannelId string

	// SponsorId is a string identifier which indicates who is sponsoring this
	// Psiphon client. One purpose of this value is to determine the home
	// pages for display. This parameter is required. This value is supplied
	// by and depends on the Psiphon Network, and is typically embedded in the
	// client binary.
	SponsorId string

	// ClientVersion is the client version number that the client reports to
	// the server. The version number refers to the host client application,
	// not the core tunnel library. One purpose of this value is to enable
	// automatic updates. This value is supplied by and depends on the Psiphon
	// Network, and is typically embedded in the client binary.
	//
	// Note that sending a ClientPlatform string which includes "windows"
	// (case insensitive) and a ClientVersion of <= 44 will cause an error in
	// processing the response to DoConnectedRequest calls.
	ClientVersion string

	// ClientPlatform is the client platform ("Windows", "Android", etc.) that
	// the client reports to the server.
	ClientPlatform string

	// TunnelWholeDevice is a flag that is passed through to the handshake
	// request for stats purposes. Set to 1 when the host application is
	// tunneling the whole device, 0 otherwise.
	TunnelWholeDevice int

	// EgressRegion is a ISO 3166-1 alpha-2 country code which indicates which
	// country to egress from. For the default, "", the best performing server
	// in any country is selected.
	EgressRegion string

	// ListenInterface specifies which interface to listen on.  If no
	// interface is provided then listen on 127.0.0.1. If 'any' is provided
	// then use 0.0.0.0. If there are multiple IP addresses on an interface
	// use the first IPv4 address.
	ListenInterface string

	// DisableLocalSocksProxy disables running the local SOCKS proxy.
	DisableLocalSocksProxy bool

	// LocalSocksProxyPort specifies a port number for the local SOCKS proxy
	// running at 127.0.0.1. For the default value, 0, the system selects a
	// free port (a notice reporting the selected port is emitted).
	LocalSocksProxyPort int

	// LocalHttpProxyPort specifies a port number for the local HTTP proxy
	// running at 127.0.0.1. For the default value, 0, the system selects a
	// free port (a notice reporting the selected port is emitted).
	LocalHttpProxyPort int

	// DisableLocalHTTPProxy disables running the local HTTP proxy.
	DisableLocalHTTPProxy bool

	// TunnelProtocol indicates which protocol to use. For the default, "",
	// all protocols are used.
	//
	// Deprecated: Use TunnelProtocols. When TunnelProtocols is not nil, this
	// parameter is ignored.
	TunnelProtocol string

	// TunnelProtocols indicates which protocols to use. Valid values include:
	// "SSH", "OSSH", "UNFRONTED-MEEK-OSSH", "UNFRONTED-MEEK-HTTPS-OSSH",
	// "UNFRONTED-MEEK-SESSION-TICKET-OSSH", "FRONTED-MEEK-OSSH", "FRONTED-
	// MEEK-HTTP-OSSH".
	//
	// For the default, an empty list, all protocols are used.
	TunnelProtocols []string

	// EstablishTunnelTimeoutSeconds specifies a time limit after which to
	// halt the core tunnel controller if no tunnel has been established. The
	// default is parameters.EstablishTunnelTimeoutSeconds.
	EstablishTunnelTimeoutSeconds *int

	// EstablishTunnelPausePeriodSeconds specifies the delay between attempts
	// to establish tunnels. Briefly pausing allows for network conditions to
	// improve and for asynchronous operations such as fetch remote server
	// list to complete. If omitted, a default value is used. This value is
	// typical overridden for testing.
	EstablishTunnelPausePeriodSeconds *int

	// ConnectionWorkerPoolSize specifies how many connection attempts to
	// attempt in parallel. If omitted of when 0, a default is used; this is
	// recommended.
	ConnectionWorkerPoolSize int

	// TunnelPoolSize specifies how many tunnels to run in parallel. Port
	// forwards are multiplexed over multiple tunnels. If omitted or when 0,
	// the default is TUNNEL_POOL_SIZE, which is recommended.
	TunnelPoolSize int

	// StaggerConnectionWorkersMilliseconds adds a specified delay before
	// making each server candidate available to connection workers. This
	// option is enabled when StaggerConnectionWorkersMilliseconds > 0.
	StaggerConnectionWorkersMilliseconds int

	// LimitMeekConnectionWorkers limits the number of concurrent connection
	// workers attempting connections with meek protocols. This option is
	// enabled when LimitMeekConnectionWorkers > 0.
	LimitMeekConnectionWorkers int

	// LimitMeekBufferSizes selects smaller buffers for meek protocols.
	LimitMeekBufferSizes bool

	// IgnoreHandshakeStatsRegexps skips compiling and using stats regexes.
	IgnoreHandshakeStatsRegexps bool

	// UpstreamProxyURL is a URL specifying an upstream proxy to use for all
	// outbound connections. The URL should include proxy type and
	// authentication information, as required. See example URLs here:
	// https://github.com/Psiphon-Labs/psiphon-tunnel-core/tree/master/psiphon/upstreamproxy
	UpstreamProxyURL string

	// CustomHeaders is a set of additional arbitrary HTTP headers that are
	// added to all plaintext HTTP requests and requests made through an HTTP
	// upstream proxy when specified by UpstreamProxyURL.
	CustomHeaders http.Header

	// Deprecated: Use CustomHeaders. When CustomHeaders is not nil, this
	// parameter is ignored.
	UpstreamProxyCustomHeaders http.Header

	// NetworkConnectivityChecker is an interface that enables tunnel-core to
	// call into the host application to check for network connectivity. See:
	// NetworkConnectivityChecker doc.
	//
	// This parameter is only applicable to library deployments.
	NetworkConnectivityChecker NetworkConnectivityChecker

	// DeviceBinder is an interface that enables tunnel-core to call into the
	// host application to bind sockets to specific devices. See: DeviceBinder
	// doc.
	//
	// This parameter is only applicable to library deployments.
	DeviceBinder DeviceBinder

	// IPv6Synthesizer is an interface that allows tunnel-core to call into
	// the host application to synthesize IPv6 addresses. See: IPv6Synthesizer
	// doc.
	//
	// This parameter is only applicable to library deployments.
	IPv6Synthesizer IPv6Synthesizer

	// DnsServerGetter is an interface that enables tunnel-core to call into
	// the host application to discover the native network DNS server
	// settings. See: DnsServerGetter doc.
	//
	// This parameter is only applicable to library deployments.
	DnsServerGetter DnsServerGetter

	// NetworkIDGetter in an interface that enables tunnel-core to call into
	// the host application to get an identifier for the host's current active
	// network. See: NetworkIDGetter doc.
	//
	// This parameter is only applicable to library deployments.
	NetworkIDGetter NetworkIDGetter

	// TransformHostNames specifies whether to use hostname transformation
	// circumvention strategies. Set to "always" to always transform, "never"
	// to never transform, and "", the default, for the default transformation
	// strategy.
	TransformHostNames string

	// TargetServerEntry is an encoded server entry. When specified, this
	// server entry is used exclusively and all other known servers are
	// ignored.
	TargetServerEntry string

	// DisableApi disables Psiphon server API calls including handshake,
	// connected, status, etc. This is used for special case temporary tunnels
	// (Windows VPN mode).
	DisableApi bool

	// TargetApiProtocol specifies whether to force use of "ssh" or "web" API
	// protocol. When blank, the default, the optimal API protocol is used.
	// Note that this capability check is not applied before the
	// "CandidateServers" count is emitted. This parameter is intended for
	// testing and debugging only.
	TargetApiProtocol string

	// RemoteServerListUrl is a URL which specifies a location to fetch out-
	// of-band server entries. This facility is used when a tunnel cannot be
	// established to known servers. This value is supplied by and depends on
	// the Psiphon Network, and is typically embedded in the client binary.
	//
	// Deprecated: Use RemoteServerListURLs. When RemoteServerListURLs is not
	// nil, this parameter is ignored.
	RemoteServerListUrl string

	// RemoteServerListURLs is list of URLs which specify locations to fetch
	// out-of-band server entries. This facility is used when a tunnel cannot
	// be established to known servers. This value is supplied by and depends
	// on the Psiphon Network, and is typically embedded in the client binary.
	// All URLs must point to the same entity with the same ETag. At least one
	// DownloadURL must have OnlyAfterAttempts = 0.
	RemoteServerListURLs parameters.DownloadURLs

	// RemoteServerListDownloadFilename specifies a target filename for
	// storing the remote server list download. Data is stored in co-located
	// files (RemoteServerListDownloadFilename.part*) to allow for resumable
	// downloading.
	RemoteServerListDownloadFilename string

	// RemoteServerListSignaturePublicKey specifies a public key that's used
	// to authenticate the remote server list payload. This value is supplied
	// by and depends on the Psiphon Network, and is typically embedded in the
	// client binary.
	RemoteServerListSignaturePublicKey string

	// DisableRemoteServerListFetcher disables fetching remote server lists.
	// This is used for special case temporary tunnels.
	DisableRemoteServerListFetcher bool

	// FetchRemoteServerListRetryPeriodMilliseconds specifies the delay before
	// resuming a remote server list download after a failure. If omitted, a
	// default value is used. This value is typical overridden for testing.
	FetchRemoteServerListRetryPeriodMilliseconds *int

	// ObfuscatedServerListRootURL is a URL which specifies the root location
	// from which to fetch obfuscated server list files. This value is
	// supplied by and depends on the Psiphon Network, and is typically
	// embedded in the client binary.
	//
	// Deprecated: Use ObfuscatedServerListRootURLs. When
	// ObfuscatedServerListRootURLs is not nil, this parameter is ignored.
	ObfuscatedServerListRootURL string

	// ObfuscatedServerListRootURLs is a list of URLs which specify root
	// locations from which to fetch obfuscated server list files. This value
	// is supplied by and depends on the Psiphon Network, and is typically
	// embedded in the client binary. All URLs must point to the same entity
	// with the same ETag. At least one DownloadURL must have
	// OnlyAfterAttempts = 0.
	ObfuscatedServerListRootURLs parameters.DownloadURLs

	// ObfuscatedServerListDownloadDirectory specifies a target directory for
	// storing the obfuscated remote server list downloads. Data is stored in
	// co-located files (<OSL filename>.part*) to allow for resumable
	// downloading.
	ObfuscatedServerListDownloadDirectory string

	// SplitTunnelRoutesURLFormat is a URL which specifies the location of a
	// routes file to use for split tunnel mode. The URL must include a
	// placeholder for the client region to be supplied. Split tunnel mode
	// uses the routes file to classify port forward destinations as foreign
	// or domestic and does not tunnel domestic destinations. Split tunnel
	// mode is on when all the SplitTunnel parameters are supplied. This value
	// is supplied by and depends on the Psiphon Network, and is typically
	// embedded in the client binary.
	SplitTunnelRoutesURLFormat string

	// SplitTunnelRoutesSignaturePublicKey specifies a public key that's used
	// to authenticate the split tunnel routes payload. This value is supplied
	// by and depends on the Psiphon Network, and is typically embedded in the
	// client binary.
	SplitTunnelRoutesSignaturePublicKey string

	// SplitTunnelDNSServer specifies a DNS server to use when resolving port
	// forward target domain names to IP addresses for classification. The DNS
	// server must support TCP requests.
	SplitTunnelDNSServer string

	// UpgradeDownloadUrl specifies a URL from which to download a host client
	// upgrade file, when one is available. The core tunnel controller
	// provides a resumable download facility which downloads this resource
	// and emits a notice when complete. This value is supplied by and depends
	// on the Psiphon Network, and is typically embedded in the client binary.
	//
	// Deprecated: Use UpgradeDownloadURLs. When UpgradeDownloadURLs is not
	// nil, this parameter is ignored.
	UpgradeDownloadUrl string

	// UpgradeDownloadURLs is list of URLs which specify locations from which
	// to download a host client upgrade file, when one is available. The core
	// tunnel controller provides a resumable download facility which
	// downloads this resource and emits a notice when complete. This value is
	// supplied by and depends on the Psiphon Network, and is typically
	// embedded in the client binary. All URLs must point to the same entity
	// with the same ETag. At least one DownloadURL must have
	// OnlyAfterAttempts = 0.
	UpgradeDownloadURLs parameters.DownloadURLs

	// UpgradeDownloadClientVersionHeader specifies the HTTP header name for
	// the entity at UpgradeDownloadURLs which specifies the client version
	// (an integer value). A HEAD request may be made to check the version
	// number available at UpgradeDownloadURLs.
	// UpgradeDownloadClientVersionHeader is required when UpgradeDownloadURLs
	// is specified.
	UpgradeDownloadClientVersionHeader string

	// UpgradeDownloadFilename is the local target filename for an upgrade
	// download. This parameter is required when UpgradeDownloadURLs (or
	// UpgradeDownloadUrl) is specified. Data is stored in co-located files
	// (UpgradeDownloadFilename.part*) to allow for resumable downloading.
	UpgradeDownloadFilename string

	// FetchUpgradeRetryPeriodMilliseconds specifies the delay before resuming
	// a client upgrade download after a failure. If omitted, a default value
	// is used. This value is typical overridden for testing.
	FetchUpgradeRetryPeriodMilliseconds *int

	// EmitBytesTransferred indicates whether to emit periodic notices showing
	// bytes sent and received.
	EmitBytesTransferred bool

	// UseIndistinguishableTLS enables use of alternative TLS profiles with a
	// less distinct fingerprint (ClientHello content) than the stock Go TLS.
	UseIndistinguishableTLS bool

	// UseTrustedCACertificatesForStockTLS toggles use of the trusted CA
	// certs, specified in TrustedCACertificatesFilename, for tunneled TLS
	// connections that expect server certificates signed with public
	// certificate authorities (currently, only upgrade downloads). This
	// option is used with stock Go TLS in cases where Go may fail to obtain a
	// list of root CAs from the operating system. Requires
	// TrustedCACertificatesFilename to be set.
	UseTrustedCACertificatesForStockTLS bool

	// TrustedCACertificatesFilename specifies a file containing trusted CA
	// certs. The file contents should be compatible with OpenSSL's
	// SSL_CTX_load_verify_locations. When specified, this enables use of
	// OpenSSL for HTTPS requests that require typical (system CA) server
	// authentication.
	TrustedCACertificatesFilename string

	// DisablePeriodicSshKeepAlive indicates whether to send an SSH keepalive
	// every 1-2 minutes, when the tunnel is idle. If the SSH keepalive times
	// out, the tunnel is considered to have failed.
	DisablePeriodicSshKeepAlive bool

	// DeviceRegion is the optional, reported region the host device is
	// running in. This input value should be a ISO 3166-1 alpha-2 country
	// code. The device region is reported to the server in the connected
	// request and recorded for Psiphon stats.
	//
	// When provided, this value may be used, pre-connection, to select
	// performance or circumvention optimization strategies for the given
	// region.
	DeviceRegion string

	// EmitDiagnosticNotices indicates whether to output notices containing
	// detailed information about the Psiphon session. As these notices may
	// contain sensitive network information, they should not be insecurely
	// distributed or displayed to users. Default is off.
	EmitDiagnosticNotices bool

	// RateLimits specify throttling configuration for the tunnel.
	RateLimits common.RateLimits

	// EmitSLOKs indicates whether to emit notices for each seeded SLOK. As
	// this could reveal user browsing activity, it's intended for debugging
	// and testing only.
	EmitSLOKs bool

	// PacketTunnelTunDeviceFileDescriptor specifies a tun device file
	// descriptor to use for running a packet tunnel. When this value is > 0,
	// a packet tunnel is established through the server and packets are
	// relayed via the tun device file descriptor. The file descriptor is
	// duped in NewController. When PacketTunnelTunDeviceFileDescriptor is
	// set, TunnelPoolSize must be 1.
	PacketTunnelTunFileDescriptor int

	// SessionID specifies a client session ID to use in the Psiphon API. The
	// session ID should be a randomly generated value that is used only for a
	// single session, which is defined as the period between a user starting
	// a Psiphon client and stopping the client.
	//
	// A session ID must be 32 hex digits (lower case). When blank, a random
	// session ID is automatically generated. Supply a session ID when a
	// single client session will cross multiple Controller instances.
	SessionID string

	// Authorizations is a list of encoded, signed access control
	// authorizations that the client has obtained and will present to the
	// server.
	Authorizations []string

	// clientParameters is the active ClientParameters with defaults, config
	// values, and, optionally, tactics applied.
	//
	// New tactics must be applied by calling Config.SetClientParameters;
	// calling clientParameters.Set directly will fail to add config values.
	clientParameters *parameters.ClientParameters
}

// LoadConfig parses and validates a JSON format Psiphon config JSON
// string and returns a Config struct populated with config values.
func LoadConfig(configJson []byte) (*Config, error) {

	var config Config
	err := json.Unmarshal(configJson, &config)
	if err != nil {
		return nil, common.ContextError(err)
	}

	// Do SetEmitDiagnosticNotices first, to ensure config file errors are emitted.

	if config.EmitDiagnosticNotices {
		SetEmitDiagnosticNotices(true)
	}

	// Promote legacy fields.

	if config.CustomHeaders == nil {
		config.CustomHeaders = config.UpstreamProxyCustomHeaders
		config.UpstreamProxyCustomHeaders = nil
	}

	if config.RemoteServerListUrl != "" && config.RemoteServerListURLs == nil {
		config.RemoteServerListURLs = promoteLegacyDownloadURL(config.RemoteServerListUrl)
	}

	if config.ObfuscatedServerListRootURL != "" && config.ObfuscatedServerListRootURLs == nil {
		config.ObfuscatedServerListRootURLs = promoteLegacyDownloadURL(config.ObfuscatedServerListRootURL)
	}

	if config.UpgradeDownloadUrl != "" && config.UpgradeDownloadURLs == nil {
		config.UpgradeDownloadURLs = promoteLegacyDownloadURL(config.UpgradeDownloadUrl)
	}

	// Supply default values.

	if config.DataStoreDirectory == "" {
		config.DataStoreDirectory, err = os.Getwd()
		if err != nil {
			return nil, common.ContextError(err)
		}
	}

	if config.ClientVersion == "" {
		config.ClientVersion = "0"
	}

	if config.TunnelPoolSize == 0 {
		config.TunnelPoolSize = TUNNEL_POOL_SIZE
	}

	// Validate config fields.

	if config.PropagationChannelId == "" {
		return nil, common.ContextError(
			errors.New("propagation channel ID is missing from the configuration file"))
	}
	if config.SponsorId == "" {
		return nil, common.ContextError(
			errors.New("sponsor ID is missing from the configuration file"))
	}

	_, err = strconv.Atoi(config.ClientVersion)
	if err != nil {
		return nil, common.ContextError(
			fmt.Errorf("invalid client version: %s", err))
	}

	if config.NetworkConnectivityChecker != nil {
		return nil, common.ContextError(
			errors.New("NetworkConnectivityChecker interface must be set at runtime"))
	}

	if config.DeviceBinder != nil {
		return nil, common.ContextError(
			errors.New("DeviceBinder interface must be set at runtime"))
	}

	if config.DnsServerGetter != nil {
		return nil, common.ContextError(
			errors.New("DnsServerGetter interface must be set at runtime"))
	}

	if !common.Contains(
		[]string{"", protocol.PSIPHON_SSH_API_PROTOCOL, protocol.PSIPHON_WEB_API_PROTOCOL},
		config.TargetApiProtocol) {

		return nil, common.ContextError(
			errors.New("invalid TargetApiProtocol"))
	}

	if !config.DisableRemoteServerListFetcher {

		if config.RemoteServerListURLs != nil {
			if config.RemoteServerListSignaturePublicKey == "" {
				return nil, common.ContextError(errors.New("missing RemoteServerListSignaturePublicKey"))
			}
			if config.RemoteServerListDownloadFilename == "" {
				return nil, common.ContextError(errors.New("missing RemoteServerListDownloadFilename"))
			}
		}

		if config.ObfuscatedServerListRootURLs != nil {
			if config.RemoteServerListSignaturePublicKey == "" {
				return nil, common.ContextError(errors.New("missing RemoteServerListSignaturePublicKey"))
			}
			if config.ObfuscatedServerListDownloadDirectory == "" {
				return nil, common.ContextError(errors.New("missing ObfuscatedServerListDownloadDirectory"))
			}
		}

	}

	if config.SplitTunnelRoutesURLFormat != "" {
		if config.SplitTunnelRoutesSignaturePublicKey == "" {
			return nil, common.ContextError(errors.New("missing SplitTunnelRoutesSignaturePublicKey"))
		}
		if config.SplitTunnelDNSServer == "" {
			return nil, common.ContextError(errors.New("missing SplitTunnelDNSServer"))
		}
	}

	if config.UpgradeDownloadURLs != nil {
		if config.UpgradeDownloadClientVersionHeader == "" {
			return nil, common.ContextError(errors.New("missing UpgradeDownloadClientVersionHeader"))
		}
		if config.UpgradeDownloadFilename == "" {
			return nil, common.ContextError(errors.New("missing UpgradeDownloadFilename"))
		}
	}

	// This constraint is expected by logic in Controller.runTunnels().

	if config.PacketTunnelTunFileDescriptor > 0 && config.TunnelPoolSize != 1 {
		return nil, common.ContextError(errors.New("packet tunnel mode requires TunnelPoolSize to be 1"))
	}

	// SessionID must be PSIPHON_API_CLIENT_SESSION_ID_LENGTH lowercase hex-encoded bytes.

	if config.SessionID == "" {
		sessionID, err := MakeSessionId()
		if err != nil {
			return nil, common.ContextError(err)
		}
		config.SessionID = sessionID
	}

	if len(config.SessionID) != 2*protocol.PSIPHON_API_CLIENT_SESSION_ID_LENGTH ||
		-1 != strings.IndexFunc(config.SessionID, func(c rune) bool {
			return !unicode.Is(unicode.ASCII_Hex_Digit, c) || unicode.IsUpper(c)
		}) {
		return nil, common.ContextError(errors.New("invalid SessionID"))
	}

	config.clientParameters, err = parameters.NewClientParameters(
		func(err error) {
			NoticeAlert("ClientParameters getValue failed: %s", err)
		})
	if err != nil {
		return nil, common.ContextError(err)
	}

	// clientParameters.Set will validate the config fields applied to parameters.

	err = config.SetClientParameters("", false, nil)
	if err != nil {
		return nil, common.ContextError(err)
	}

	return &config, nil
}

// GetClientParameters returns a snapshot of the current client parameters.
func (config *Config) GetClientParameters() *parameters.ClientParametersSnapshot {
	return config.clientParameters.Get()
}

// SetClientParameters resets Config.clientParameters to the default values,
// applies any config file values, and then applies the input parameters (from
// tactics, etc.)
//
// Set skipOnError to false when initially applying only config values, as
// this will validate the values and should fail. Set skipOnError to true when
// applying tactics to ignore invalid or unknown parameter values from tactics.
//
// In the case of applying tactics, do not call Config.clientParameters.Set
// directly as this will not first apply config values.
//
// If there is an error, the existing Config.clientParameters are left
// entirely unmodified.
func (config *Config) SetClientParameters(tag string, skipOnError bool, applyParameters map[string]interface{}) error {

	parameters := []map[string]interface{}{config.makeConfigParameters()}
	if applyParameters != nil {
		parameters = append(parameters, applyParameters)
	}

	counts, err := config.clientParameters.Set(tag, skipOnError, parameters...)
	if err != nil {
		return common.ContextError(err)
	}

	NoticeInfo("applied %v parameters with tag '%s'", counts, tag)

	return nil
}

func (config *Config) makeConfigParameters() map[string]interface{} {

	// Build set of config values to apply to parameters.
	//
	// Note: names of some config fields such as
	// StaggerConnectionWorkersMilliseconds and LimitMeekBufferSizes have
	// changed in the parameters. The existing config fields are retained for
	// backwards compatibility.

	applyParameters := make(map[string]interface{})

	if len(config.TunnelProtocols) > 0 {
		applyParameters[parameters.LimitTunnelProtocols] = protocol.TunnelProtocols(config.TunnelProtocols)
	} else if config.TunnelProtocol != "" {
		applyParameters[parameters.LimitTunnelProtocols] = protocol.TunnelProtocols{config.TunnelProtocol}
	}

	if config.EstablishTunnelTimeoutSeconds != nil {
		applyParameters[parameters.EstablishTunnelTimeout] = fmt.Sprintf("%ds", *config.EstablishTunnelTimeoutSeconds)
	}

	if config.EstablishTunnelPausePeriodSeconds != nil {
		applyParameters[parameters.EstablishTunnelPausePeriod] = fmt.Sprintf("%ds", *config.EstablishTunnelPausePeriodSeconds)
	}

	if config.ConnectionWorkerPoolSize != 0 {
		applyParameters[parameters.ConnectionWorkerPoolSize] = config.ConnectionWorkerPoolSize
	}

	if config.StaggerConnectionWorkersMilliseconds > 0 {
		applyParameters[parameters.StaggerConnectionWorkersPeriod] = fmt.Sprintf("%dms", config.StaggerConnectionWorkersMilliseconds)
	}

	if config.LimitMeekConnectionWorkers > 0 {
		applyParameters[parameters.LimitMeekConnectionWorkers] = config.LimitMeekConnectionWorkers
	}

	applyParameters[parameters.MeekLimitBufferSizes] = config.LimitMeekBufferSizes

	applyParameters[parameters.IgnoreHandshakeStatsRegexps] = config.IgnoreHandshakeStatsRegexps

	if config.EstablishTunnelTimeoutSeconds != nil {
		applyParameters[parameters.EstablishTunnelTimeout] = fmt.Sprintf("%ds", *config.EstablishTunnelTimeoutSeconds)
	}

	if config.FetchRemoteServerListRetryPeriodMilliseconds != nil {
		applyParameters[parameters.FetchRemoteServerListRetryPeriod] = fmt.Sprintf("%dms", *config.FetchRemoteServerListRetryPeriodMilliseconds)
	}

	if config.FetchUpgradeRetryPeriodMilliseconds != nil {
		applyParameters[parameters.FetchUpgradeRetryPeriod] = fmt.Sprintf("%dms", *config.FetchUpgradeRetryPeriodMilliseconds)
	}

	if !config.DisableRemoteServerListFetcher {

		if config.RemoteServerListURLs != nil {
			applyParameters[parameters.RemoteServerListSignaturePublicKey] = config.RemoteServerListSignaturePublicKey
			applyParameters[parameters.RemoteServerListURLs] = config.RemoteServerListURLs
		}

		if config.ObfuscatedServerListRootURLs != nil {
			applyParameters[parameters.RemoteServerListSignaturePublicKey] = config.RemoteServerListSignaturePublicKey
			applyParameters[parameters.ObfuscatedServerListRootURLs] = config.ObfuscatedServerListRootURLs
		}

	}

	applyParameters[parameters.SplitTunnelRoutesURLFormat] = config.SplitTunnelRoutesURLFormat
	applyParameters[parameters.SplitTunnelRoutesSignaturePublicKey] = config.SplitTunnelRoutesSignaturePublicKey
	applyParameters[parameters.SplitTunnelDNSServer] = config.SplitTunnelDNSServer

	if config.UpgradeDownloadURLs != nil {
		applyParameters[parameters.UpgradeDownloadClientVersionHeader] = config.UpgradeDownloadClientVersionHeader
		applyParameters[parameters.UpgradeDownloadURLs] = config.UpgradeDownloadURLs
	}

	applyParameters[parameters.TunnelRateLimits] = config.RateLimits

	return applyParameters
}

func promoteLegacyDownloadURL(URL string) parameters.DownloadURLs {
	downloadURLs := make(parameters.DownloadURLs, 1)
	downloadURLs[0] = &parameters.DownloadURL{
		URL:               base64.StdEncoding.EncodeToString([]byte(URL)),
		SkipVerify:        false,
		OnlyAfterAttempts: 0,
	}
	return downloadURLs
}
