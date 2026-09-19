const Protocols = {
    VMESS: 'vmess',
    VLESS: 'vless',
    TROJAN: 'trojan',
    SHADOWSOCKS: 'shadowsocks',
    DOKODEMO: 'dokodemo-door',
    MTPROTO: 'mtproto',
    SOCKS: 'socks',
    HTTP: 'http',
    MIXED: 'mixed',
    TUNNEL: 'tunnel',
};

const VmessMethods = {
    AES_128_GCM: 'aes-128-gcm',
    CHACHA20_POLY1305: 'chacha20-poly1305',
    AUTO: 'auto',
    NONE: 'none',
};

const SSMethods = {
    // AES_256_CFB: 'aes-256-cfb',
    // AES_128_CFB: 'aes-128-cfb',
    // CHACHA20: 'chacha20',
    // CHACHA20_IETF: 'chacha20-ietf',
    CHACHA20_POLY1305: 'chacha20-poly1305',
    AES_256_GCM: 'aes-256-gcm',
    AES_128_GCM: 'aes-128-gcm',
    BLAKE3_AES_128_GCM_2022: '2022-blake3-aes-128-gcm',
    BLAKE3_AES_256_GCM_2022: '2022-blake3-aes-256-gcm',
    BLAKE3_CHACHA20_POLY1305_2022: '2022-blake3-chacha20-poly1305',
};

const SSLegacyMethods = [
    SSMethods.CHACHA20_POLY1305,
    SSMethods.AES_256_GCM,
    SSMethods.AES_128_GCM,
];

const SS2022Methods = [
    SSMethods.BLAKE3_AES_128_GCM_2022,
    SSMethods.BLAKE3_AES_256_GCM_2022,
    SSMethods.BLAKE3_CHACHA20_POLY1305_2022,
];

const ShadowsocksCredential = Object.freeze({
    keyLength(method) {
        switch (String(method || '').trim().toLowerCase()) {
            case SSMethods.BLAKE3_AES_128_GCM_2022:
                return 16;
            case SSMethods.BLAKE3_AES_256_GCM_2022:
            case SSMethods.BLAKE3_CHACHA20_POLY1305_2022:
                return 32;
            default:
                return 0;
        }
    },
    is2022(method) {
        return this.keyLength(method) > 0;
    },
    isValid2022Key(method, password) {
        const expected = this.keyLength(method);
        if (!expected || typeof password !== 'string' || !password) {
            return false;
        }
        try {
            const decoded = atob(password);
            return decoded.length === expected && btoa(decoded) === password;
        } catch (e) {
            return false;
        }
    },
});

const RULE_IP = {
    PRIVATE: 'geoip:private',
    CN: 'geoip:cn',
};

const RULE_DOMAIN = {
    ADS: 'geosite:category-ads',
    ADS_ALL: 'geosite:category-ads-all',
    CN: 'geosite:cn',
    GOOGLE: 'geosite:google',
    FACEBOOK: 'geosite:facebook',
    SPEEDTEST: 'geosite:speedtest',
};

const FLOW_CONTROL = {
    VISION: "xtls-rprx-vision",
};

Object.freeze(Protocols);
Object.freeze(VmessMethods);
Object.freeze(SSMethods);
Object.freeze(SSLegacyMethods);
Object.freeze(SS2022Methods);
Object.freeze(RULE_IP);
Object.freeze(RULE_DOMAIN);
Object.freeze(FLOW_CONTROL);

class XrayCommonClass {

    static toJsonArray(arr) {
        return arr.map(obj => obj.toJson());
    }

    static fromJson() {
        return new XrayCommonClass();
    }

    toJson() {
        return this;
    }

    toString(format=true) {
        return format ? JSON.stringify(this.toJson(), null, 2) : JSON.stringify(this.toJson());
    }

    static toHeaders(v2Headers) {
        let newHeaders = [];
        if (v2Headers) {
            Object.keys(v2Headers).forEach(key => {
                let values = v2Headers[key];
                if (typeof(values) === 'string') {
                    newHeaders.push({ name: key, value: values });
                } else {
                    for (let i = 0; i < values.length; ++i) {
                        newHeaders.push({ name: key, value: values[i] });
                    }
                }
            });
        }
        return newHeaders;
    }

    static toV2Headers(headers, arr=true) {
        let v2Headers = {};
        for (let i = 0; i < headers.length; ++i) {
            let name = headers[i].name;
            let value = headers[i].value;
            if (ObjectUtil.isEmpty(name) || ObjectUtil.isEmpty(value)) {
                continue;
            }
            if (!(name in v2Headers)) {
                v2Headers[name] = arr ? [value] : value;
            } else {
                if (arr) {
                    v2Headers[name].push(value);
                } else {
                    v2Headers[name] = value;
                }
            }
        }
        return v2Headers;
    }
}

class TcpStreamSettings extends XrayCommonClass {
    constructor(type='none',
                request=new TcpStreamSettings.TcpRequest(),
                response=new TcpStreamSettings.TcpResponse(),
                ) {
        super();
        this.type = type;
        this.request = request;
        this.response = response;
    }

    static fromJson(json={}) {
        let header = json.header;
        if (!header) {
            header = {};
        }
        return new TcpStreamSettings(
            header.type,
            TcpStreamSettings.TcpRequest.fromJson(header.request),
            TcpStreamSettings.TcpResponse.fromJson(header.response),
        );
    }

    toJson() {
        return {
            header: {
                type: this.type,
                request: this.type === 'http' ? this.request.toJson() : undefined,
                response: this.type === 'http' ? this.response.toJson() : undefined,
            },
        };
    }
}

TcpStreamSettings.TcpRequest = class extends XrayCommonClass {
    constructor(version='1.1',
                method='GET',
                path=['/'],
                headers=[],
    ) {
        super();
        this.version = version;
        this.method = method;
        this.path = path.length === 0 ? ['/'] : path;
        this.headers = headers;
    }

    addPath(path) {
        this.path.push(path);
    }

    removePath(index) {
        this.path.splice(index, 1);
    }

    addHeader(name, value) {
        this.headers.push({ name: name, value: value });
    }

    getHeader(name) {
        for (const header of this.headers) {
            if (header.name.toLowerCase() === name.toLowerCase()) {
                return header.value;
            }
        }
        return null;
    }

    removeHeader(index) {
        this.headers.splice(index, 1);
    }

    static fromJson(json={}) {
        return new TcpStreamSettings.TcpRequest(
            json.version,
            json.method,
            json.path,
            XrayCommonClass.toHeaders(json.headers),
        );
    }

    toJson() {
        return {
            method: this.method,
            path: ObjectUtil.clone(this.path),
            headers: XrayCommonClass.toV2Headers(this.headers),
        };
    }
};

TcpStreamSettings.TcpResponse = class extends XrayCommonClass {
    constructor(version='1.1',
                status='200',
                reason='OK',
                headers=[],
    ) {
        super();
        this.version = version;
        this.status = status;
        this.reason = reason;
        this.headers = headers;
    }

    addHeader(name, value) {
        this.headers.push({ name: name, value: value });
    }

    removeHeader(index) {
        this.headers.splice(index, 1);
    }

    static fromJson(json={}) {
        return new TcpStreamSettings.TcpResponse(
            json.version,
            json.status,
            json.reason,
            XrayCommonClass.toHeaders(json.headers),
        );
    }

    toJson() {
        return {
            version: this.version,
            status: this.status,
            reason: this.reason,
            headers: XrayCommonClass.toV2Headers(this.headers),
        };
    }
};

class KcpStreamSettings extends XrayCommonClass {
    constructor(mtu=1350, tti=20,
                uplinkCapacity=5,
                downlinkCapacity=20,
                congestion=false,
                readBufferSize=2,
                writeBufferSize=2,
                type='none',
                seed=RandomUtil.randomSeq(10),
                ) {
        super();
        this.mtu = mtu;
        this.tti = tti;
        this.upCap = uplinkCapacity;
        this.downCap = downlinkCapacity;
        this.congestion = congestion;
        this.readBuffer = readBufferSize;
        this.writeBuffer = writeBufferSize;
        this.type = type;
        this.seed = seed;
    }

    static fromJson(json={}) {
        return new KcpStreamSettings(
            json.mtu,
            json.tti,
            json.uplinkCapacity,
            json.downlinkCapacity,
            json.congestion,
            json.readBufferSize,
            json.writeBufferSize,
            ObjectUtil.isEmpty(json.header) ? 'none' : json.header.type,
            json.seed,
        );
    }

    toJson() {
        return {
            mtu: this.mtu,
            tti: this.tti,
            uplinkCapacity: this.upCap,
            downlinkCapacity: this.downCap,
            congestion: this.congestion,
            readBufferSize: this.readBuffer,
            writeBufferSize: this.writeBuffer,
            header: {
                type: this.type,
            },
            seed: this.seed,
        };
    }
}

class WsStreamSettings extends XrayCommonClass {
    constructor(path='/', headers=[]) {
        super();
        this.path = path;
        this.headers = headers;
    }

    addHeader(name, value) {
        this.headers.push({ name: name, value: value });
    }

    getHeader(name) {
        for (const header of this.headers) {
            if (header.name.toLowerCase() === name.toLowerCase()) {
                return header.value;
            }
        }
        return null;
    }

    removeHeader(index) {
        this.headers.splice(index, 1);
    }

    static fromJson(json={}) {
        return new WsStreamSettings(
            json.path,
            XrayCommonClass.toHeaders(json.headers),
        );
    }

    toJson() {
        return {
            path: this.path,
            headers: XrayCommonClass.toV2Headers(this.headers, false),
        };
    }
}

class HttpStreamSettings extends XrayCommonClass {
    constructor(path='/', host=['']) {
        super();
        this.path = path;
        this.host = host.length === 0 ? [''] : host;
    }

    addHost(host) {
        this.host.push(host);
    }

    removeHost(index) {
        this.host.splice(index, 1);
    }

    static fromJson(json={}) {
        return new HttpStreamSettings(json.path, json.host);
    }

    toJson() {
        let host = [];
        for (let i = 0; i < this.host.length; ++i) {
            if (!ObjectUtil.isEmpty(this.host[i])) {
                host.push(this.host[i]);
            }
        }
        return {
            path: this.path,
            host: host,
        }
    }
}

class QuicStreamSettings extends XrayCommonClass {
    constructor(security=VmessMethods.NONE,
                key='', type='none') {
        super();
        this.security = security;
        this.key = key;
        this.type = type;
    }

    static fromJson(json={}) {
        return new QuicStreamSettings(
            json.security,
            json.key,
            json.header ? json.header.type : 'none',
        );
    }

    toJson() {
        return {
            security: this.security,
            key: this.key,
            header: {
                type: this.type,
            }
        }
    }
}

class GrpcStreamSettings extends XrayCommonClass {
    constructor(serviceName="") {
        super();
        this.serviceName = serviceName;
    }

    static fromJson(json={}) {
        return new GrpcStreamSettings(json.serviceName);
    }

    toJson() {
        return {
            serviceName: this.serviceName,
        }
    }
}

class TlsStreamSettings extends XrayCommonClass {
    constructor(serverName='',
                certificates=[new TlsStreamSettings.Cert()]) {
        super();
        this.server = serverName;
        this.certs = certificates;
    }

    addCert(cert) {
        this.certs.push(cert);
    }

    removeCert(index) {
        this.certs.splice(index, 1);
    }

    static fromJson(json={}) {
        let certs;
        if (!ObjectUtil.isEmpty(json.certificates)) {
            certs = json.certificates.map(cert => TlsStreamSettings.Cert.fromJson(cert));
        }
        return new TlsStreamSettings(
            json.serverName,
            certs,
        );
    }

    toJson() {
        return {
            serverName: this.server,
            certificates: TlsStreamSettings.toJsonArray(this.certs),
        };
    }
}

TlsStreamSettings.Cert = class extends XrayCommonClass {
    constructor(useFile=true, certificateFile='', keyFile='', certificate='', key='') {
        super();
        this.useFile = useFile;
        this.certFile = certificateFile;
        this.keyFile = keyFile;
        this.cert = certificate instanceof Array ? certificate.join('\n') : certificate;
        this.key = key instanceof Array ? key.join('\n') : key;
    }

    static fromJson(json={}) {
        if ('certificateFile' in json && 'keyFile' in json) {
            return new TlsStreamSettings.Cert(
                true,
                json.certificateFile,
                json.keyFile,
            );
        } else {
            return new TlsStreamSettings.Cert(
                false, '', '',
                json.certificate.join('\n'),
                json.key.join('\n'),
            );
        }
    }

    toJson() {
        if (this.useFile) {
            return {
                certificateFile: this.certFile,
                keyFile: this.keyFile,
            };
        } else {
            return {
                certificate: this.cert.split('\n'),
                key: this.key.split('\n'),
            };
        }
    }
};

class RealityStreamSettings extends XrayCommonClass {
    constructor(dest='',
                serverNames=[''],
                privateKey='',
                shortIds=[''],
                fingerprint='chrome',
                publicKey='',
                spiderX='') {
        super();
        this.dest = dest;
        this.serverNames = serverNames;
        this.privateKey = privateKey;
        this.shortIds = shortIds;
        this.fingerprint = fingerprint;
        this.publicKey = publicKey;
        this.spiderX = spiderX;
    }

    static splitList(value) {
        if (Array.isArray(value)) {
            return value;
        }
        if (ObjectUtil.isEmpty(value)) {
            return [''];
        }
        return String(value)
            .split(',')
            .map(item => item.trim())
            .filter(item => !ObjectUtil.isEmpty(item));
    }

    get serverNamesText() {
        return this.serverNames.join(',');
    }

    set serverNamesText(value) {
        this.serverNames = RealityStreamSettings.splitList(value);
    }

    get shortIdsText() {
        return this.shortIds.join(',');
    }

    set shortIdsText(value) {
        this.shortIds = RealityStreamSettings.splitList(value);
    }

    static fromJson(json={}) {
        return new RealityStreamSettings(
            json.dest || '',
            Array.isArray(json.serverNames) ? json.serverNames : [''],
            json.privateKey || '',
            Array.isArray(json.shortIds) ? json.shortIds : [''],
            json.fingerprint || 'chrome',
            json.publicKey || '',
            json.spiderX || '',
        );
    }

    toJson() {
        if (ObjectUtil.isEmpty(this.fingerprint)) {
            this.fingerprint = 'chrome';
        }
        if (ObjectUtil.isEmpty(this.spiderX)) {
            this.spiderX = '/';
        }
        return {
            dest: this.dest,
            serverNames: this.serverNames.filter(serverName => !ObjectUtil.isEmpty(serverName)),
            privateKey: this.privateKey,
            shortIds: this.shortIds.filter(shortId => !ObjectUtil.isEmpty(shortId)),
            fingerprint: this.fingerprint,
            publicKey: this.publicKey,
            spiderX: this.spiderX,
        };
    }
}

class StreamSettings extends XrayCommonClass {
    constructor(network='tcp',
                security='none',
                tlsSettings=new TlsStreamSettings(),
                realitySettings=new RealityStreamSettings(),
                tcpSettings=new TcpStreamSettings(),
                kcpSettings=new KcpStreamSettings(),
                wsSettings=new WsStreamSettings(),
                httpSettings=new HttpStreamSettings(),
                quicSettings=new QuicStreamSettings(),
                grpcSettings=new GrpcStreamSettings(),
                ) {
        super();
        this.network = network;
        this.security = security;
        this.tls = tlsSettings;
        this.reality = realitySettings;
        this.tcp = tcpSettings;
        this.kcp = kcpSettings;
        this.ws = wsSettings;
        this.http = httpSettings;
        this.quic = quicSettings;
        this.grpc = grpcSettings;
    }

    get isTls() {
        return this.security === 'tls';
    }

    set isTls(isTls) {
        if (isTls) {
            this.security = 'tls';
        } else {
            this.security = 'none';
        }
    }

    get isXTls() {
        return this.security === "xtls";
    }

    set isXTls(isXTls) {
        if (isXTls) {
            this.security = 'xtls';
        } else {
            this.security = 'none';
        }
    }

    get isReality() {
        return this.security === 'reality';
    }

    set isReality(isReality) {
        if (isReality) {
            this.security = 'reality';
        } else {
            this.security = 'none';
        }
    }

    static fromJson(json={}) {
        let tls;
        if (json.security === "xtls") {
            tls = TlsStreamSettings.fromJson(json.xtlsSettings);
        } else {
            tls = TlsStreamSettings.fromJson(json.tlsSettings);
        }
        return new StreamSettings(
            json.network,
            json.security,
            tls,
            RealityStreamSettings.fromJson(json.realitySettings),
            TcpStreamSettings.fromJson(json.tcpSettings),
            KcpStreamSettings.fromJson(json.kcpSettings),
            WsStreamSettings.fromJson(json.wsSettings),
            HttpStreamSettings.fromJson(json.httpSettings),
            QuicStreamSettings.fromJson(json.quicSettings),
            GrpcStreamSettings.fromJson(json.grpcSettings),
        );
    }

    toJson() {
        const network = this.network;
        return {
            network: network,
            security: this.security,
            tlsSettings: this.isTls ? this.tls.toJson() : undefined,
            xtlsSettings: this.isXTls ? this.tls.toJson() : undefined,
            realitySettings: this.isReality ? this.reality.toJson() : undefined,
            tcpSettings: network === 'tcp' ? this.tcp.toJson() : undefined,
            kcpSettings: network === 'kcp' ? this.kcp.toJson() : undefined,
            wsSettings: network === 'ws' ? this.ws.toJson() : undefined,
            httpSettings: network === 'http' ? this.http.toJson() : undefined,
            quicSettings: network === 'quic' ? this.quic.toJson() : undefined,
            grpcSettings: network === 'grpc' ? this.grpc.toJson() : undefined,
        };
    }
}

class Sniffing extends XrayCommonClass {
    constructor(enabled=false, destOverride=['http', 'tls'], metadataOnly=false, routeOnly=false) {
        super();
        this.enabled = enabled;
        this.destOverride = destOverride;
        this.metadataOnly = metadataOnly;
        this.routeOnly = routeOnly;
    }

    static fromJson(json={}) {
        let destOverride = ObjectUtil.clone(json.destOverride);
        if (!ObjectUtil.isEmpty(destOverride) && !ObjectUtil.isArrEmpty(destOverride)) {
            if (ObjectUtil.isEmpty(destOverride[0])) {
                destOverride = ['http', 'tls'];
            }
        }
        return new Sniffing(
            !!json.enabled,
            destOverride,
            !!json.metadataOnly,
            !!json.routeOnly,
        );
    }
}

class Inbound extends XrayCommonClass {
    constructor(port=RandomUtil.randomIntRange(10000, 60000),
                listen='',
                protocol=Protocols.VMESS,
                settings=null,
                streamSettings=new StreamSettings(),
                tag='',
                sniffing=new Sniffing(),
                ) {
        super();
        this.port = port;
        this.listen = listen;
        this._protocol = protocol;
        this.settings = ObjectUtil.isEmpty(settings) ? Inbound.Settings.getSettings(protocol) : settings;
        this.stream = streamSettings;
        this.tag = tag;
        this.sniffing = sniffing;
    }

    get protocol() {
        return this._protocol;
    }

    set protocol(protocol) {
        this._protocol = protocol;
        this.settings = Inbound.Settings.getSettings(protocol);
        this.stream.security = 'none';
    }

    get tls() {
        return this.stream.security === 'tls';
    }

    set tls(isTls) {
        if (isTls) {
            this.stream.security = 'tls';
        } else {
            this.stream.security = 'none';
        }
    }

    get xtls() {
        return this.stream.security === 'xtls';
    }

    set xtls(isXTls) {
        if (isXTls) {
            this.stream.security = 'xtls';
        } else {
            this.stream.security = 'none';
        }
    }

    get reality() {
        return this.stream.security === 'reality';
    }

    set reality(isReality) {
        if (isReality) {
            this.stream.security = 'reality';
        } else {
            this.stream.security = 'none';
        }
    }

    get network() {
        return this.stream.network;
    }

    set network(network) {
        this.stream.network = network;
    }

    get isTcp() {
        return this.network === "tcp";
    }

    get isWs() {
        return this.network === "ws";
    }

    get isKcp() {
        return this.network === "kcp";
    }

    get isQuic() {
        return this.network === "quic"
    }

    get isGrpc() {
        return this.network === "grpc";
    }

    get isH2() {
        return this.network === "http";
    }

    // VMess & VLess
    get uuid() {
        switch (this.protocol) {
            case Protocols.VMESS:
                return this.settings.vmesses[0].id;
            case Protocols.VLESS:
                return this.settings.vlesses[0].id;
            default:
                return "";
        }
    }

    // VLess & Trojan
    get flow() {
        switch (this.protocol) {
            case Protocols.VLESS:
                return this.settings.vlesses[0].flow;
            default:
                return "";
        }
    }

    // VMess
    get alterId() {
        switch (this.protocol) {
            case Protocols.VMESS:
                return this.settings.vmesses[0].alterId;
            default:
                return "";
        }
    }

    // Socks & HTTP & Mixed
    get username() {
        switch (this.protocol) {
            case Protocols.SOCKS:
            case Protocols.HTTP:
            case Protocols.MIXED:
                if (Array.isArray(this.settings.accounts) && this.settings.accounts.length > 0) {
                    return this.settings.accounts[0].user;
                }
                return "";
            default:
                return "";
        }
    }

    // Trojan & Shadowsocks & Socks & HTTP & Mixed
    get password() {
        switch (this.protocol) {
            case Protocols.TROJAN:
                return this.settings.clients[0].password;
            case Protocols.SHADOWSOCKS:
                return this.settings.password;
            case Protocols.SOCKS:
            case Protocols.HTTP:
            case Protocols.MIXED:
                if (Array.isArray(this.settings.accounts) && this.settings.accounts.length > 0) {
                    return this.settings.accounts[0].pass;
                }
                return "";
            default:
                return "";
        }
    }

    // Shadowsocks
    get method() {
        switch (this.protocol) {
            case Protocols.SHADOWSOCKS:
                return this.settings.method;
            default:
                return "";
        }
    }

    get serverName() {
        if (this.stream.isTls || this.stream.isXTls) {
            return this.stream.tls.server;
        }
        if (this.stream.isReality && Array.isArray(this.stream.reality.serverNames)) {
            return this.stream.reality.serverNames[0] || "";
        }
        return "";
    }

    get host() {
        if (this.isTcp) {
            return this.stream.tcp.request.getHeader("Host");
        } else if (this.isWs) {
            return this.stream.ws.getHeader("Host");
        } else if (this.isH2) {
            return this.stream.http.host[0];
        }
        return null;
    }

    get path() {
        if (this.isTcp) {
            return this.stream.tcp.request.path[0];
        } else if (this.isWs) {
            return this.stream.ws.path;
        } else if (this.isH2) {
            return this.stream.http.path[0];
        }
        return null;
    }

    get quicSecurity() {
        return this.stream.quic.security;
    }

    get quicKey() {
        return this.stream.quic.key;
    }

    get quicType() {
        return this.stream.quic.type;
    }

    get kcpType() {
        return this.stream.kcp.type;
    }

    get kcpSeed() {
        return this.stream.kcp.seed;
    }

    get serviceName() {
        return this.stream.grpc.serviceName;
    }

    canEnableTls() {
        switch (this.protocol) {
            case Protocols.VMESS:
            case Protocols.VLESS:
            case Protocols.TROJAN:
            case Protocols.SHADOWSOCKS:
                break;
            default:
                return false;
        }

        switch (this.network) {
            case "tcp":
            case "ws":
            case "http":
            case "quic":
            case "grpc":
                return true;
            default:
                return false;
        }
    }

    canSetTls() {
        return this.canEnableTls();
    }

    canEnableXTls() {
        switch (this.protocol) {
            case Protocols.VLESS:
                break;
            default:
                return false;
        }
        return this.network === "tcp";
    }

    canEnableReality() {
        if (this.protocol !== Protocols.VLESS) {
            return false;
        }
        return this.network === "tcp" || this.network === "grpc";
    }

    canEnableStream() {
        switch (this.protocol) {
            case Protocols.VMESS:
            case Protocols.VLESS:
            case Protocols.TROJAN:
            case Protocols.SHADOWSOCKS:
                return true;
            default:
                return false;
        }
    }

    canSniffing() {
        switch (this.protocol) {
            case Protocols.VMESS:
            case Protocols.VLESS:
            case Protocols.TROJAN:
            case Protocols.SHADOWSOCKS:
                return true;
            default:
                return false;
        }
    }

    reset() {
        this.port = RandomUtil.randomIntRange(10000, 60000);
        this.listen = '';
        this.protocol = Protocols.VMESS;
        this.settings = Inbound.Settings.getSettings(Protocols.VMESS);
        this.stream = new StreamSettings();
        this.tag = '';
        this.sniffing = new Sniffing();
    }

    static fromShareLink(link) {
        if (ObjectUtil.isEmpty(link)) {
            throw new Error('分享链接为空');
        }
        const value = link.trim();
        if (value.startsWith('vmess://')) {
            return Inbound.parseVMessLink(value);
        }
        if (value.startsWith('vless://')) {
            return Inbound.parseVLESSLink(value);
        }
        if (value.startsWith('trojan://')) {
            return Inbound.parseTrojanLink(value);
        }
        if (value.startsWith('ss://')) {
            return Inbound.parseSSLink(value);
        }
        throw new Error('不支持的分享链接协议');
    }

    static decodeShareBase64(value) {
        let text = value.trim().replace(/-/g, '+').replace(/_/g, '/');
        while (text.length % 4 !== 0) {
            text += '=';
        }
        return Base64.decode(text);
    }

    static parseSharePort(port) {
        const value = parseInt(port, 10);
        if (!Number.isInteger(value) || value < 1 || value > 65535) {
            throw new Error('分享链接端口无效');
        }
        return value;
    }

    static decodeShareValue(value) {
        if (ObjectUtil.isEmpty(value)) {
            return '';
        }
        try {
            return decodeURIComponent(value);
        } catch (e) {
            return value;
        }
    }

    static parseShareUrl(link, scheme) {
        const prefix = scheme + '://';
        if (!link.startsWith(prefix)) {
            throw new Error('分享链接协议无效');
        }

        let raw = link.substring(prefix.length);
        let fragment = '';
        const hashIndex = raw.indexOf('#');
        if (hashIndex >= 0) {
            fragment = raw.substring(hashIndex + 1);
            raw = raw.substring(0, hashIndex);
        }

        let queryString = '';
        const queryIndex = raw.indexOf('?');
        if (queryIndex >= 0) {
            queryString = raw.substring(queryIndex + 1);
            raw = raw.substring(0, queryIndex);
        }

        const atIndex = raw.lastIndexOf('@');
        if (atIndex < 0) {
            throw new Error('分享链接认证信息无效');
        }

        const userInfo = raw.substring(0, atIndex);
        const hostPort = raw.substring(atIndex + 1);
        const server = Inbound.parseShareHostPort(hostPort);
        return {
            userInfo: userInfo,
            host: server.host,
            port: server.port,
            query: new URLSearchParams(queryString),
            fragment: fragment,
        };
    }

    static normalizeShareNetwork(network) {
        if (ObjectUtil.isEmpty(network)) {
            return 'tcp';
        }
        switch (String(network).toLowerCase()) {
            case 'tcp':
            case 'kcp':
            case 'ws':
            case 'quic':
            case 'grpc':
                return String(network).toLowerCase();
            case 'http':
            case 'h2':
                return 'http';
            case 'httpupgrade':
                throw new Error('当前不支持 HTTPUpgrade 传输');
            default:
                throw new Error('不支持的传输类型: ' + network);
        }
    }

    static setShareHeader(headers, name, value) {
        if (ObjectUtil.isEmpty(value)) {
            return;
        }
        const index = headers.findIndex(header => header.name.toLowerCase() === name.toLowerCase());
        if (index >= 0) {
            headers[index].value = value;
        } else {
            headers.push({ name: name, value: value });
        }
    }

    static applyShareStream(inbound, options={}) {
        const network = Inbound.normalizeShareNetwork(options.network || options.type);
        inbound.stream.network = network;

        let security = (options.security || '').toLowerCase();
        if (security === 'reality') {
            inbound.stream.security = 'reality';
            inbound.stream.reality.serverNames = RealityStreamSettings.splitList(options.sni || options.serverName);
            inbound.stream.reality.fingerprint = options.fingerprint || options.fp || 'chrome';
            inbound.stream.reality.publicKey = options.publicKey || options.pbk || '';
            inbound.stream.reality.shortIds = RealityStreamSettings.splitList(options.shortId || options.sid);
            inbound.stream.reality.spiderX = options.spiderX || options.spx || '/';
        } else if (security === 'tls' || security === 'xtls') {
            inbound.stream.security = security;
            inbound.stream.tls.server = options.sni || options.serverName || '';
        } else {
            inbound.stream.security = 'none';
        }

        switch (network) {
            case 'tcp':
                if ((options.headerType || options.tcpType) === 'http') {
                    inbound.stream.tcp.type = 'http';
                    if (!ObjectUtil.isEmpty(options.path)) {
                        inbound.stream.tcp.request.path = String(options.path).split(',');
                    }
                    Inbound.setShareHeader(inbound.stream.tcp.request.headers, 'Host', options.host);
                }
                break;
            case 'kcp':
                inbound.stream.kcp.type = options.headerType || options.kcpType || 'none';
                inbound.stream.kcp.seed = options.seed || '';
                break;
            case 'ws':
                inbound.stream.ws.path = options.path || '/';
                Inbound.setShareHeader(inbound.stream.ws.headers, 'Host', options.host);
                break;
            case 'http':
                inbound.stream.http.path = options.path || '/';
                inbound.stream.http.host = ObjectUtil.isEmpty(options.host) ? [''] : String(options.host).split(',');
                break;
            case 'quic':
                inbound.stream.quic.security = options.quicSecurity || options.host || VmessMethods.NONE;
                inbound.stream.quic.key = options.key || options.path || '';
                inbound.stream.quic.type = options.headerType || options.quicType || 'none';
                break;
            case 'grpc':
                inbound.stream.grpc.serviceName = options.serviceName || options.path || '';
                break;
        }
    }

    static parseVMessLink(link) {
        const body = link.substring('vmess://'.length).split(/[?#]/)[0];
        const config = JSON.parse(Inbound.decodeShareBase64(body));
        const port = Inbound.parseSharePort(config.port);
        const inbound = new Inbound(port, '', Protocols.VMESS);
        inbound.settings.vmesses[0].id = config.id || config.uuid || '';
        inbound.settings.vmesses[0].alterId = parseInt(config.aid || config.alterId || 0, 10);
        Inbound.applyShareStream(inbound, {
            network: config.net || config.type,
            security: config.tls === 'tls' || config.tls === 'xtls' ? config.tls : 'none',
            sni: config.sni || config.serverName,
            tcpType: config.type,
            headerType: config.type,
            host: config.host,
            path: config.path,
        });
        return {
            inbound: inbound,
            remark: config.ps || '',
            address: config.add || '',
        };
    }

    static parseVLESSLink(link) {
        const parsed = Inbound.parseShareUrl(link, 'vless');
        const query = parsed.query;
        const security = (query.get('security') || 'none').toLowerCase();
        const inbound = new Inbound(parsed.port, '', Protocols.VLESS);
        inbound.settings.vlesses[0].id = Inbound.decodeShareValue(parsed.userInfo);
        inbound.settings.decryption = query.get('encryption') || 'none';
        inbound.settings.vlesses[0].flow = query.get('flow') || '';
        if (security === 'reality') {
            const network = Inbound.normalizeShareNetwork(query.get('type') || 'tcp');
            if (network !== 'tcp' && network !== 'grpc') {
                throw new Error('Reality 分享链接仅支持 tcp 或 grpc 传输');
            }
        }
        Inbound.applyShareStream(inbound, {
            network: query.get('type') || 'tcp',
            security: security,
            sni: query.get('sni'),
            fingerprint: query.get('fp') || query.get('fingerprint'),
            publicKey: query.get('pbk') || query.get('publicKey'),
            shortId: query.get('sid') || query.get('shortId'),
            spiderX: query.get('spx') || query.get('spiderX'),
            headerType: query.get('headerType'),
            host: query.get('host'),
            path: query.get('path'),
            seed: query.get('seed'),
            quicSecurity: query.get('quicSecurity'),
            key: query.get('key'),
            serviceName: query.get('serviceName'),
        });
        return {
            inbound: inbound,
            remark: Inbound.decodeShareValue(parsed.fragment),
            address: parsed.host,
            requiresServerPrivateKey: security === 'reality',
        };
    }

    static parseTrojanLink(link) {
        const parsed = Inbound.parseShareUrl(link, 'trojan');
        const query = parsed.query;
        const security = (query.get('security') || 'tls').toLowerCase();
        if (security === 'reality') {
            throw new Error('当前不支持 Reality 分享链接导入');
        }
        const inbound = new Inbound(parsed.port, '', Protocols.TROJAN);
        inbound.settings.clients[0].password = Inbound.decodeShareValue(parsed.userInfo);
        Inbound.applyShareStream(inbound, {
            network: query.get('type') || 'tcp',
            security: security,
            sni: query.get('sni'),
            headerType: query.get('headerType'),
            host: query.get('host'),
            path: query.get('path'),
            seed: query.get('seed'),
            quicSecurity: query.get('quicSecurity'),
            key: query.get('key'),
            serviceName: query.get('serviceName'),
        });
        return {
            inbound: inbound,
            remark: Inbound.decodeShareValue(parsed.fragment),
            address: parsed.host,
        };
    }

    static parseShareHostPort(value) {
        let host = '';
        let port = '';
        if (value.startsWith('[')) {
            const index = value.indexOf(']');
            host = value.substring(1, index);
            port = value.substring(index + 2);
        } else {
            const index = value.lastIndexOf(':');
            host = value.substring(0, index);
            port = value.substring(index + 1);
        }
        if (ObjectUtil.isEmpty(host)) {
            throw new Error('分享链接地址无效');
        }
        return {
            host: host,
            port: Inbound.parseSharePort(port),
        };
    }

    static parseSSLink(link) {
        const raw = link.substring('ss://'.length);
        const hashIndex = raw.indexOf('#');
        const withoutHash = hashIndex >= 0 ? raw.substring(0, hashIndex) : raw;
        const remark = hashIndex >= 0 ? Inbound.decodeShareValue(raw.substring(hashIndex + 1)) : '';
        const queryIndex = withoutHash.indexOf('?');
        const body = queryIndex >= 0 ? withoutHash.substring(0, queryIndex) : withoutHash;
        const query = queryIndex >= 0 ? new URLSearchParams(withoutHash.substring(queryIndex + 1)) : new URLSearchParams();
        if (!ObjectUtil.isEmpty(query.get('plugin'))) {
            throw new Error('当前不支持 Shadowsocks plugin 分享链接导入');
        }

        let userInfo = '';
        let hostPort = '';
        const atIndex = body.lastIndexOf('@');
        if (atIndex >= 0) {
            userInfo = body.substring(0, atIndex);
            hostPort = body.substring(atIndex + 1);
            const decodedUserInfo = Inbound.decodeShareValue(userInfo);
            if (decodedUserInfo.includes(':')) {
                userInfo = decodedUserInfo;
            } else {
                userInfo = Inbound.decodeShareBase64(userInfo);
            }
        } else {
            const decoded = Inbound.decodeShareBase64(body);
            const decodedAtIndex = decoded.lastIndexOf('@');
            if (decodedAtIndex < 0) {
                throw new Error('Shadowsocks 分享链接格式无效');
            }
            userInfo = decoded.substring(0, decodedAtIndex);
            hostPort = decoded.substring(decodedAtIndex + 1);
        }

        const colonIndex = userInfo.indexOf(':');
        if (colonIndex < 0) {
            throw new Error('Shadowsocks 分享链接认证信息无效');
        }
        const method = userInfo.substring(0, colonIndex);
        const password = userInfo.substring(colonIndex + 1);
        const server = Inbound.parseShareHostPort(hostPort);
        const inbound = new Inbound(server.port, '', Protocols.SHADOWSOCKS);
        inbound.settings.method = method;
        inbound.settings.password = password;
        return {
            inbound: inbound,
            remark: remark,
            address: server.host,
        };
    }

    validateBasic() {
        const port = Number(this.port);
        if (!Number.isInteger(port) || port < 1 || port > 65535) {
            return '端口必须是 1-65535 的整数';
        }

        switch (this.protocol) {
            case Protocols.VMESS:
                if (ObjectUtil.isEmpty(this.settings.vmesses[0].id)) {
                    return 'VMess UUID 不能为空';
                }
                break;
            case Protocols.VLESS:
                if (ObjectUtil.isEmpty(this.settings.vlesses[0].id)) {
                    return 'VLESS UUID 不能为空';
                }
                if (this.reality) {
                    if (!this.canEnableReality()) {
                        return 'Reality 仅支持 tcp 或 grpc 传输';
                    }
                    if (ObjectUtil.isEmpty(this.stream.reality.dest)) {
                        return 'Reality dest 不能为空';
                    }
                    if (ObjectUtil.isArrEmpty(this.stream.reality.serverNames.filter(serverName => !ObjectUtil.isEmpty(serverName)))) {
                        return 'Reality serverNames 不能为空';
                    }
                if (ObjectUtil.isEmpty(this.stream.reality.privateKey)) {
                    return 'Reality privateKey 不能为空';
                }
                    const shortIds = this.stream.reality.shortIds.filter(shortId => !ObjectUtil.isEmpty(shortId));
                    if (ObjectUtil.isArrEmpty(shortIds)) {
                        return 'Reality shortIds 不能为空';
                    }
                    for (const shortId of shortIds) {
                        if (!/^[0-9a-fA-F]{2,16}$/.test(shortId) || shortId.length % 2 !== 0) {
                            return 'Reality shortId 必须是 2-16 位偶数长度十六进制字符串';
                        }
                    }
                    if (ObjectUtil.isEmpty(this.stream.reality.fingerprint)) {
                        this.stream.reality.fingerprint = 'chrome';
                    }
                    if (ObjectUtil.isEmpty(this.stream.reality.spiderX)) {
                        this.stream.reality.spiderX = '/';
                    }
                    if (!ObjectUtil.isEmpty(this.stream.reality.spiderX) && !this.stream.reality.spiderX.startsWith('/')) {
                        return 'Reality spiderX 必须以 / 开头';
                    }
                }
                break;
            case Protocols.TROJAN:
                if (ObjectUtil.isEmpty(this.settings.clients[0].password)) {
                    return 'Trojan 密码不能为空';
                }
                break;
            case Protocols.SHADOWSOCKS:
                if (ObjectUtil.isEmpty(this.settings.method)) {
                    return 'Shadowsocks 加密方式不能为空';
                }
                if (ObjectUtil.isEmpty(this.settings.password)) {
                    return 'Shadowsocks 密码不能为空';
                }
                if (ShadowsocksCredential.is2022(this.settings.method)
                    && !ShadowsocksCredential.isValid2022Key(this.settings.method, this.settings.password)) {
                    const keyLength = ShadowsocksCredential.keyLength(this.settings.method);
                    return `Shadowsocks 2022 密钥必须是标准 Base64，解码后为 ${keyLength} 字节`;
                }
                break;
        }
        return '';
    }

    genVmessLink(address='', remark='', overrideAddress=false) {
        if (this.protocol !== Protocols.VMESS) {
            return '';
        }
        let network = this.stream.network;
        let type = 'none';
        let host = '';
        let path = '';
        if (network === 'tcp') {
            let tcp = this.stream.tcp;
            type = tcp.type;
            if (type === 'http') {
                let request = tcp.request;
                path = request.path.join(',');
                let index = request.headers.findIndex(header => header.name.toLowerCase() === 'host');
                if (index >= 0) {
                    host = request.headers[index].value;
                }
            }
        } else if (network === 'kcp') {
            let kcp = this.stream.kcp;
            type = kcp.type;
            path = kcp.seed;
        } else if (network === 'ws') {
            let ws = this.stream.ws;
            path = ws.path;
            let index = ws.headers.findIndex(header => header.name.toLowerCase() === 'host');
            if (index >= 0) {
                host = ws.headers[index].value;
            }
        } else if (network === 'http') {
            network = 'h2';
            path = this.stream.http.path;
            host = this.stream.http.host.join(',');
        } else if (network === 'quic') {
            type = this.stream.quic.type;
            host = this.stream.quic.security;
            path = this.stream.quic.key;
        } else if (network === 'grpc') {
            path = this.stream.grpc.serviceName;
        }

        if (overrideAddress) {
            address = normalizeShareAddress(address);
        } else if (this.stream.security === 'tls') {
            if (!ObjectUtil.isEmpty(this.stream.tls.server)) {
                address = this.stream.tls.server;
            }
        }

        let obj = {
            v: '2',
            ps: remark,
            add: address,
            port: this.port,
            id: this.settings.vmesses[0].id,
            aid: this.settings.vmesses[0].alterId,
            scy: VmessMethods.AUTO,
            net: network,
            type: type,
            host: host,
            path: path,
            tls: this.stream.security,
            sni: this.stream.security === 'tls' ? this.stream.tls.server : '',
        };
        return 'vmess://' + base64(JSON.stringify(obj, null, 2));
    }

    genVLESSLink(address = '', remark='', overrideAddress=false) {
        if (overrideAddress) {
            address = normalizeShareAddress(address, true);
        }
        const settings = this.settings;
        const uuid = settings.vlesses[0].id;
        const port = this.port;
        const type = this.stream.network;
        const params = new Map();
        params.set("encryption", settings.decryption || "none");
        params.set("type", this.stream.network);
        if (this.xtls) {
            params.set("security", "xtls");
        } else {
            params.set("security", this.stream.security);
        }
        switch (type) {
            case "tcp":
                const tcp = this.stream.tcp;
                if (tcp.type === 'http') {
                    const request = tcp.request;
                    params.set("path", request.path.join(','));
                    const index = request.headers.findIndex(header => header.name.toLowerCase() === 'host');
                    if (index >= 0) {
                        const host = request.headers[index].value;
                        params.set("host", host);
                    }
                }
                break;
            case "kcp":
                const kcp = this.stream.kcp;
                params.set("headerType", kcp.type);
                params.set("seed", kcp.seed);
                break;
            case "ws":
                const ws = this.stream.ws;
                params.set("path", ws.path);
                const index = ws.headers.findIndex(header => header.name.toLowerCase() === 'host');
                if (index >= 0) {
                    const host = ws.headers[index].value;
                    params.set("host", host);
                }
                break;
            case "http":
                const http = this.stream.http;
                params.set("path", http.path);
                params.set("host", http.host);
                break;
            case "quic":
                const quic = this.stream.quic;
                params.set("quicSecurity", quic.security);
                params.set("key", quic.key);
                params.set("headerType", quic.type);
                break;
            case "grpc":
                const grpc = this.stream.grpc;
                params.set("serviceName", grpc.serviceName);
                break;
        }

        if (this.reality) {
            if (type !== 'tcp' && type !== 'grpc') {
                throw new Error('Reality 分享仅支持 tcp 或 grpc 传输');
            }
            const reality = this.stream.reality;
            const serverName = Array.isArray(reality.serverNames) ? reality.serverNames[0] : '';
            const shortId = Array.isArray(reality.shortIds) ? reality.shortIds[0] : '';
            if (ObjectUtil.isEmpty(reality.publicKey)) {
                throw new Error('Reality publicKey 不能为空，请先生成 X25519 密钥');
            }
            params.set("security", "reality");
            params.set("sni", serverName || address);
            params.set("fp", reality.fingerprint || 'chrome');
            params.set("pbk", reality.publicKey);
            params.set("sid", shortId || '');
            if (!ObjectUtil.isEmpty(reality.spiderX)) {
                params.set("spx", reality.spiderX);
            }
        } else if (this.stream.security === 'tls') {
            if (!ObjectUtil.isEmpty(this.stream.tls.server)) {
                if (overrideAddress) {
                    params.set("sni", this.stream.tls.server);
                } else {
                    address = this.stream.tls.server;
                    params.set("sni", address);
                }
            }
        }

        if (this.xtls) {
            params.set("flow", this.settings.vlesses[0].flow);
        } else if (!ObjectUtil.isEmpty(this.settings.vlesses[0].flow)) {
            params.set("flow", this.settings.vlesses[0].flow);
        }

        const link = `vless://${uuid}@${address}:${port}`;
        const url = new URL(link);
        for (const [key, value] of params) {
            url.searchParams.set(key, value)
        }
        url.hash = remark;
        return url.toString();
    }

    genSSLink(address='', remark='', overrideAddress=false) {
        let settings = this.settings;
        const server = this.stream.tls.server;
        if (overrideAddress) {
            address = normalizeShareAddress(address, true);
        } else if (!ObjectUtil.isEmpty(server)) {
            address = server;
        }
        return 'ss://' + safeBase64(settings.method + ':' + settings.password + '@' + address + ':' + this.port)
            + '#' + encodeURIComponent(remark);
    }

    genTrojanLink(address='', remark='', overrideAddress=false) {
        let settings = this.settings;
        if (overrideAddress) {
            address = normalizeShareAddress(address, true);
        } else if (this.stream.security === 'tls' || this.stream.security === 'xtls') {
            if (!ObjectUtil.isEmpty(this.stream.tls.server)) {
                address = this.stream.tls.server;
            }
        }

        const params = new Map();
        params.set("security", this.stream.security);
        params.set("type", this.stream.network);
        switch (this.stream.network) {
            case "tcp":
                if (this.stream.tcp.type === 'http') {
                    const request = this.stream.tcp.request;
                    params.set("path", request.path.join(','));
                    const index = request.headers.findIndex(header => header.name.toLowerCase() === 'host');
                    if (index >= 0) {
                        params.set("host", request.headers[index].value);
                    }
                }
                break;
            case "ws":
                params.set("path", this.stream.ws.path);
                const wsHostIndex = this.stream.ws.headers.findIndex(header => header.name.toLowerCase() === 'host');
                if (wsHostIndex >= 0) {
                    params.set("host", this.stream.ws.headers[wsHostIndex].value);
                }
                break;
            case "grpc":
                params.set("serviceName", this.stream.grpc.serviceName);
                break;
            case "http":
                params.set("path", this.stream.http.path);
                params.set("host", this.stream.http.host);
                break;
            case "kcp":
                params.set("headerType", this.stream.kcp.type);
                params.set("seed", this.stream.kcp.seed);
                break;
            case "quic":
                params.set("quicSecurity", this.stream.quic.security);
                params.set("key", this.stream.quic.key);
                params.set("headerType", this.stream.quic.type);
                break;
        }
        if (this.stream.security === 'tls' || this.stream.security === 'xtls') {
            if (!ObjectUtil.isEmpty(this.stream.tls.server)) {
                params.set("sni", this.stream.tls.server);
            }
        }

        const link = `trojan://${encodeURIComponent(settings.clients[0].password)}@${address}:${this.port}`;
        const url = new URL(link);
        for (const [key, value] of params) {
            url.searchParams.set(key, value);
        }
        url.hash = remark;
        return url.toString();
    }

    genLink(address='', remark='', overrideAddress=false) {
        switch (this.protocol) {
            case Protocols.VMESS: return this.genVmessLink(address, remark, overrideAddress);
            case Protocols.VLESS: return this.genVLESSLink(address, remark, overrideAddress);
            case Protocols.SHADOWSOCKS: return this.genSSLink(address, remark, overrideAddress);
            case Protocols.TROJAN: return this.genTrojanLink(address, remark, overrideAddress);
            default: return '';
        }
    }

    static fromJson(json={}) {
        return new Inbound(
            json.port,
            json.listen,
            json.protocol,
            Inbound.Settings.fromJson(json.protocol, json.settings),
            StreamSettings.fromJson(json.streamSettings),
            json.tag,
            Sniffing.fromJson(json.sniffing),
        )
    }

    toJson() {
        let streamSettings;
        if (this.canEnableStream() || this.protocol === Protocols.TROJAN) {
            streamSettings = this.stream.toJson();
        }
        return {
            port: this.port,
            listen: this.listen,
            protocol: this.protocol,
            settings: this.settings instanceof XrayCommonClass ? this.settings.toJson() : this.settings,
            streamSettings: streamSettings,
            tag: this.tag,
            sniffing: this.sniffing.toJson(),
        };
    }
}

Inbound.Settings = class extends XrayCommonClass {
    constructor(protocol) {
        super();
        this.protocol = protocol;
    }

    static getSettings(protocol) {
        switch (protocol) {
            case Protocols.VMESS: return new Inbound.VmessSettings(protocol);
            case Protocols.VLESS: return new Inbound.VLESSSettings(protocol);
            case Protocols.TROJAN: return new Inbound.TrojanSettings(protocol);
            case Protocols.SHADOWSOCKS: return new Inbound.ShadowsocksSettings(protocol);
            case Protocols.DOKODEMO: return new Inbound.DokodemoSettings(protocol);
            case Protocols.MTPROTO: return new Inbound.MtprotoSettings(protocol);
            case Protocols.SOCKS: return new Inbound.SocksSettings(protocol);
            case Protocols.HTTP: return new Inbound.HttpSettings(protocol);
            case Protocols.MIXED: return new Inbound.MixedSettings(protocol);
            case Protocols.TUNNEL: return new Inbound.TunnelSettings(protocol);
            default: return null;
        }
    }

    static fromJson(protocol, json) {
        switch (protocol) {
            case Protocols.VMESS: return Inbound.VmessSettings.fromJson(json);
            case Protocols.VLESS: return Inbound.VLESSSettings.fromJson(json);
            case Protocols.TROJAN: return Inbound.TrojanSettings.fromJson(json);
            case Protocols.SHADOWSOCKS: return Inbound.ShadowsocksSettings.fromJson(json);
            case Protocols.DOKODEMO: return Inbound.DokodemoSettings.fromJson(json);
            case Protocols.MTPROTO: return Inbound.MtprotoSettings.fromJson(json);
            case Protocols.SOCKS: return Inbound.SocksSettings.fromJson(json);
            case Protocols.HTTP: return Inbound.HttpSettings.fromJson(json);
            case Protocols.MIXED: return Inbound.MixedSettings.fromJson(json);
            case Protocols.TUNNEL: return Inbound.TunnelSettings.fromJson(json);
            default: return null;
        }
    }

    toJson() {
        return {};
    }
};

Inbound.VmessSettings = class extends Inbound.Settings {
    constructor(protocol,
                vmesses=[new Inbound.VmessSettings.Vmess()],
                disableInsecureEncryption=false) {
        super(protocol);
        this.vmesses = vmesses;
        this.disableInsecure = disableInsecureEncryption;
    }

    indexOfVmessById(id) {
        return this.vmesses.findIndex(vmess => vmess.id === id);
    }

    addVmess(vmess) {
        if (this.indexOfVmessById(vmess.id) >= 0) {
            return false;
        }
        this.vmesses.push(vmess);
    }

    delVmess(vmess) {
        const i = this.indexOfVmessById(vmess.id);
        if (i >= 0) {
            this.vmesses.splice(i, 1);
        }
    }

    static fromJson(json={}) {
        return new Inbound.VmessSettings(
            Protocols.VMESS,
            json.clients.map(client => Inbound.VmessSettings.Vmess.fromJson(client)),
            ObjectUtil.isEmpty(json.disableInsecureEncryption) ? false : json.disableInsecureEncryption,
        );
    }

    toJson() {
        return {
            clients: Inbound.VmessSettings.toJsonArray(this.vmesses),
            disableInsecureEncryption: this.disableInsecure,
        };
    }
};
Inbound.VmessSettings.Vmess = class extends XrayCommonClass {
    constructor(id=RandomUtil.randomUUID(), alterId=0) {
        super();
        this.id = id;
        this.alterId = alterId;
    }

    static fromJson(json={}) {
        return new Inbound.VmessSettings.Vmess(
            json.id,
            json.alterId,
        );
    }
};

Inbound.VLESSSettings = class extends Inbound.Settings {
    constructor(protocol,
                vlesses=[new Inbound.VLESSSettings.VLESS()],
                decryption='none',
                fallbacks=[],) {
        super(protocol);
        this.vlesses = vlesses;
        this.decryption = decryption;
        this.fallbacks = fallbacks;
    }

    addFallback() {
        this.fallbacks.push(new Inbound.VLESSSettings.Fallback());
    }

    delFallback(index) {
        this.fallbacks.splice(index, 1);
    }

    static fromJson(json={}) {
        return new Inbound.VLESSSettings(
            Protocols.VLESS,
            json.clients.map(client => Inbound.VLESSSettings.VLESS.fromJson(client)),
            json.decryption,
            Inbound.VLESSSettings.Fallback.fromJson(json.fallbacks),
        );
    }

    toJson() {
        return {
            clients: Inbound.VLESSSettings.toJsonArray(this.vlesses),
            decryption: this.decryption,
            fallbacks: Inbound.VLESSSettings.toJsonArray(this.fallbacks),
        };
    }
};
Inbound.VLESSSettings.VLESS = class extends XrayCommonClass {

    constructor(id=RandomUtil.randomUUID(), flow='') {
        super();
        this.id = id;
        this.flow = flow;
    }

    static fromJson(json={}) {
        return new Inbound.VLESSSettings.VLESS(
            json.id,
            json.flow,
        );
    }
};
Inbound.VLESSSettings.Fallback = class extends XrayCommonClass {
    constructor(name="", alpn='', path='', dest='', xver=0) {
        super();
        this.name = name;
        this.alpn = alpn;
        this.path = path;
        this.dest = dest;
        this.xver = xver;
    }

    toJson() {
        let xver = this.xver;
        if (!Number.isInteger(xver)) {
            xver = 0;
        }
        return {
            name: this.name,
            alpn: this.alpn,
            path: this.path,
            dest: this.dest,
            xver: xver,
        }
    }

    static fromJson(json=[]) {
        const fallbacks = [];
        for (let fallback of json) {
            fallbacks.push(new Inbound.VLESSSettings.Fallback(
                fallback.name,
                fallback.alpn,
                fallback.path,
                fallback.dest,
                fallback.xver,
            ))
        }
        return fallbacks;
    }
};

Inbound.TrojanSettings = class extends Inbound.Settings {
    constructor(protocol,
                clients=[new Inbound.TrojanSettings.Client()],
                fallbacks=[],) {
        super(protocol);
        this.clients = clients;
        this.fallbacks = fallbacks;
    }

    addTrojanFallback() {
        this.fallbacks.push(new Inbound.TrojanSettings.Fallback());
    }

    delTrojanFallback(index) {
        this.fallbacks.splice(index, 1);
    }

    toJson() {
        return {
            clients: Inbound.TrojanSettings.toJsonArray(this.clients),
            fallbacks: Inbound.TrojanSettings.toJsonArray(this.fallbacks),
        };
    }

    static fromJson(json={}) {
        const clients = [];
        for (const c of json.clients) {
            clients.push(Inbound.TrojanSettings.Client.fromJson(c));
        }
        return new Inbound.TrojanSettings(
            Protocols.TROJAN,
            clients,
            Inbound.TrojanSettings.Fallback.fromJson(json.fallbacks),);
    }
};
Inbound.TrojanSettings.Client = class extends XrayCommonClass {
    constructor(password=RandomUtil.randomSeq(10)) {
        super();
        this.password = password;
    }

    toJson() {
        return {
            password: this.password,
        };
    }

    static fromJson(json={}) {
        return new Inbound.TrojanSettings.Client(
            json.password,
        );
    }

};

Inbound.TrojanSettings.Fallback = class extends XrayCommonClass {
    constructor(name="", alpn='', path='', dest='', xver=0) {
        super();
        this.name = name;
        this.alpn = alpn;
        this.path = path;
        this.dest = dest;
        this.xver = xver;
    }

    toJson() {
        let xver = this.xver;
        if (!Number.isInteger(xver)) {
            xver = 0;
        }
        return {
            name: this.name,
            alpn: this.alpn,
            path: this.path,
            dest: this.dest,
            xver: xver,
        }
    }

    static fromJson(json=[]) {
        const fallbacks = [];
        for (let fallback of json) {
            fallbacks.push(new Inbound.TrojanSettings.Fallback(
                fallback.name,
                fallback.alpn,
                fallback.path,
                fallback.dest,
                fallback.xver,
            ))
        }
        return fallbacks;
    }
};

Inbound.ShadowsocksSettings = class extends Inbound.Settings {
    constructor(protocol,
                method=SSMethods.AES_256_GCM,
                password=RandomUtil.randomSeq(10),
                network='tcp,udp'
    ) {
        super(protocol);
        this.method = method;
        this.password = password;
        this.network = network;
    }

    static fromJson(json={}) {
        return new Inbound.ShadowsocksSettings(
            Protocols.SHADOWSOCKS,
            json.method,
            json.password,
            json.network,
        );
    }

    toJson() {
        return {
            method: this.method,
            password: this.password,
            network: this.network,
        };
    }
};

Inbound.DokodemoSettings = class extends Inbound.Settings {
    constructor(protocol, address='127.0.0.1', port=53, network='tcp,udp') {
        super(protocol);
        this.address = Inbound.DokodemoSettings.normalizeAddress(address);
        this.port = Inbound.DokodemoSettings.normalizePort(port, 53);
        this.network = Inbound.DokodemoSettings.normalizeNetwork(network);
    }

    static normalizeAddress(address) {
        if (ObjectUtil.isEmpty(address)) {
            return '127.0.0.1';
        }
        const normalized = String(address).trim();
        return normalized === '' ? '127.0.0.1' : normalized;
    }

    static normalizePort(port, fallback=53) {
        const parsed = Number(port);
        if (!Number.isInteger(parsed) || parsed < 0 || parsed > 65535) {
            return fallback;
        }
        return parsed;
    }

    static normalizeNetwork(network) {
        switch (network) {
            case 'tcp':
            case 'udp':
            case 'tcp,udp':
                return network;
            default:
                return 'tcp,udp';
        }
    }

    static fromJson(json={}) {
        return new Inbound.DokodemoSettings(
            Protocols.DOKODEMO,
            json.address,
            json.port,
            json.network,
        );
    }

    toJson() {
        const address = Inbound.DokodemoSettings.normalizeAddress(this.address);
        const port = Inbound.DokodemoSettings.normalizePort(this.port, 53);
        const network = Inbound.DokodemoSettings.normalizeNetwork(this.network);
        return {
            address: address,
            port: port,
            network: network,
        };
    }
};

Inbound.TunnelSettings = class extends Inbound.DokodemoSettings {
    constructor(protocol, address='127.0.0.1', port=53, network='tcp,udp') {
        super(protocol, address, port, network);
    }

    static fromJson(json={}) {
        return new Inbound.TunnelSettings(
            Protocols.TUNNEL,
            json.address,
            json.port,
            json.network,
        );
    }
};

Inbound.MtprotoSettings = class extends Inbound.Settings {
    constructor(protocol, users=[new Inbound.MtprotoSettings.MtUser()]) {
        super(protocol);
        this.users = users;
    }

    static fromJson(json={}) {
        return new Inbound.MtprotoSettings(
            Protocols.MTPROTO,
            json.users.map(user => Inbound.MtprotoSettings.MtUser.fromJson(user)),
        );
    }

    toJson() {
        return {
            users: XrayCommonClass.toJsonArray(this.users),
        };
    }
};
Inbound.MtprotoSettings.MtUser = class extends XrayCommonClass {
    constructor(secret=RandomUtil.randomMTSecret()) {
        super();
        this.secret = secret;
    }

    static fromJson(json={}) {
        return new Inbound.MtprotoSettings.MtUser(json.secret);
    }
};

Inbound.SocksSettings = class extends Inbound.Settings {
    constructor(protocol, auth='noauth', accounts=[new Inbound.SocksSettings.SocksAccount()], udp=true, ip='127.0.0.1') {
        super(protocol);
        this.auth = auth === 'password' ? 'password' : 'noauth';
        this.accounts = Array.isArray(accounts) && accounts.length > 0
            ? accounts
            : [new Inbound.SocksSettings.SocksAccount()];
        this.udp = udp;
        this.ip = ObjectUtil.isEmpty(ip) ? '127.0.0.1' : ip;
    }

    addAccount(account) {
        this.accounts.push(account);
    }

    delAccount(index) {
        this.accounts.splice(index, 1);
    }

    static parseUdp(value, defaultValue=true) {
        if (typeof value === 'boolean') {
            return value;
        }
        if (typeof value === 'string') {
            const normalized = value.trim().toLowerCase();
            if (['true', '1', 'yes', 'on'].includes(normalized)) {
                return true;
            }
            if (['false', '0', 'no', 'off', ''].includes(normalized)) {
                return false;
            }
        }
        if (typeof value === 'number') {
            return value !== 0;
        }
        return defaultValue;
    }

    static fromJson(json={}) {
        const hasAuth = Object.prototype.hasOwnProperty.call(json, 'auth');
        const hasAccounts = Array.isArray(json.accounts) && json.accounts.length > 0;
        const auth = hasAuth
            ? (json.auth === 'password' ? 'password' : 'noauth')
            : (hasAccounts ? 'password' : 'noauth');
        let accounts = [new Inbound.SocksSettings.SocksAccount()];
        if (auth === 'password' && hasAccounts) {
            accounts = json.accounts.map(
                account => Inbound.SocksSettings.SocksAccount.fromJson(account)
            );
        }
        return new Inbound.SocksSettings(
            Protocols.SOCKS,
            auth,
            accounts,
            Inbound.SocksSettings.parseUdp(json.udp, true),
            ObjectUtil.isEmpty(json.ip) ? '127.0.0.1' : json.ip,
        );
    }

    toJson() {
        if (!Array.isArray(this.accounts) || this.accounts.length === 0) {
            this.accounts = [new Inbound.SocksSettings.SocksAccount()];
        }
        return {
            auth: this.auth,
            accounts: this.auth === 'password' ? this.accounts.map(account => account.toJson()) : undefined,
            udp: this.udp,
            ip: this.ip,
        };
    }
};
Inbound.SocksSettings.SocksAccount = class extends XrayCommonClass {
    constructor(user=RandomUtil.randomSeq(10), pass=RandomUtil.randomSeq(10)) {
        super();
        this.user = user;
        this.pass = pass;
    }

    static fromJson(json={}) {
        return new Inbound.SocksSettings.SocksAccount(json.user, json.pass);
    }
};

Inbound.MixedSettings = class extends Inbound.SocksSettings {
    constructor(protocol, auth='noauth', accounts=[new Inbound.SocksSettings.SocksAccount()], udp=true, ip='127.0.0.1') {
        super(protocol, auth, accounts, udp, ip);
    }

    static fromJson(json={}) {
        const hasAuth = Object.prototype.hasOwnProperty.call(json, 'auth');
        const hasAccounts = Array.isArray(json.accounts) && json.accounts.length > 0;
        const auth = hasAuth
            ? (json.auth === 'password' ? 'password' : 'noauth')
            : (hasAccounts ? 'password' : 'noauth');
        let accounts = [new Inbound.SocksSettings.SocksAccount()];
        if (auth === 'password' && hasAccounts) {
            accounts = json.accounts.map(
                account => Inbound.SocksSettings.SocksAccount.fromJson(account)
            );
        }
        return new Inbound.MixedSettings(
            Protocols.MIXED,
            auth,
            accounts,
            Inbound.SocksSettings.parseUdp(json.udp, true),
            ObjectUtil.isEmpty(json.ip) ? '127.0.0.1' : json.ip,
        );
    }
};

Inbound.HttpSettings = class extends Inbound.Settings {
    constructor(protocol, auth='noauth', accounts=[new Inbound.HttpSettings.HttpAccount()]) {
        super(protocol);
        this.auth = auth === 'password' ? 'password' : 'noauth';
        this.accounts = Array.isArray(accounts) && accounts.length > 0
            ? accounts
            : [new Inbound.HttpSettings.HttpAccount()];
    }

    addAccount(account) {
        this.accounts.push(account);
    }

    delAccount(index) {
        this.accounts.splice(index, 1);
    }

    static fromJson(json={}) {
        const auth = Array.isArray(json.accounts) && json.accounts.length > 0 ? 'password' : 'noauth';
        let accounts = [new Inbound.HttpSettings.HttpAccount()];
        if (auth === 'password') {
            accounts = json.accounts.map(account => Inbound.HttpSettings.HttpAccount.fromJson(account));
        }
        return new Inbound.HttpSettings(
            Protocols.HTTP,
            auth,
            accounts,
        );
    }

    toJson() {
        if (this.auth === 'password' && (!Array.isArray(this.accounts) || this.accounts.length === 0)) {
            this.accounts = [new Inbound.HttpSettings.HttpAccount()];
        }
        return {
            accounts: this.auth === 'password' ? Inbound.HttpSettings.toJsonArray(this.accounts) : undefined,
        };
    }
};

Inbound.HttpSettings.HttpAccount = class extends XrayCommonClass {
    constructor(user=RandomUtil.randomSeq(10), pass=RandomUtil.randomSeq(10)) {
        super();
        this.user = user;
        this.pass = pass;
    }

    static fromJson(json={}) {
        return new Inbound.HttpSettings.HttpAccount(json.user, json.pass);
    }
};
