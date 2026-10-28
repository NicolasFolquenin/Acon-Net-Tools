export namespace main {
	
	export class Config {
	    node_id: string;
	    location: string;
	    server_url: string;
	    api_token: string;
	    interval_minutes: number;
	    ping_target: string;
	    sync_with_central: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.node_id = source["node_id"];
	        this.location = source["location"];
	        this.server_url = source["server_url"];
	        this.api_token = source["api_token"];
	        this.interval_minutes = source["interval_minutes"];
	        this.ping_target = source["ping_target"];
	        this.sync_with_central = source["sync_with_central"];
	    }
	}
	export class DNSDomainResult {
	    domain: string;
	    resolved_ip: string;
	    duration_ms: number;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DNSDomainResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.resolved_ip = source["resolved_ip"];
	        this.duration_ms = source["duration_ms"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class DNSResult {
	    domains: DNSDomainResult[];
	    avg_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new DNSResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domains = this.convertValues(source["domains"], DNSDomainResult);
	        this.avg_ms = source["avg_ms"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HTTPCheckResult {
	    url: string;
	    status_code: number;
	    duration_ms: number;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new HTTPCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.status_code = source["status_code"];
	        this.duration_ms = source["duration_ms"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class TracerouteHop {
	    hop_num: number;
	    ip: string;
	    latency_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new TracerouteHop(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hop_num = source["hop_num"];
	        this.ip = source["ip"];
	        this.latency_ms = source["latency_ms"];
	    }
	}
	export class TracerouteResult {
	    target: string;
	    total_hops: number;
	    hops: TracerouteHop[];
	    success: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TracerouteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.total_hops = source["total_hops"];
	        this.hops = this.convertValues(source["hops"], TracerouteHop);
	        this.success = source["success"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PacketLossResult {
	    target: string;
	    sent: number;
	    received: number;
	    loss_percent: number;
	    jitter_ms: number;
	    min_latency_ms: number;
	    avg_latency_ms: number;
	    max_latency_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new PacketLossResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.sent = source["sent"];
	        this.received = source["received"];
	        this.loss_percent = source["loss_percent"];
	        this.jitter_ms = source["jitter_ms"];
	        this.min_latency_ms = source["min_latency_ms"];
	        this.avg_latency_ms = source["avg_latency_ms"];
	        this.max_latency_ms = source["max_latency_ms"];
	    }
	}
	export class ModularResults {
	    dns?: DNSResult;
	    packet_loss?: PacketLossResult;
	    http_check?: HTTPCheckResult;
	    traceroute?: TracerouteResult;
	
	    static createFrom(source: any = {}) {
	        return new ModularResults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dns = this.convertValues(source["dns"], DNSResult);
	        this.packet_loss = this.convertValues(source["packet_loss"], PacketLossResult);
	        this.http_check = this.convertValues(source["http_check"], HTTPCheckResult);
	        this.traceroute = this.convertValues(source["traceroute"], TracerouteResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SwitchInfo {
	    switch_name: string;
	    switch_port: string;
	    vlan?: number;
	    switch_model?: string;
	    protocol?: string;
	    // Go type: time
	    discovered_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new SwitchInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.switch_name = source["switch_name"];
	        this.switch_port = source["switch_port"];
	        this.vlan = source["vlan"];
	        this.switch_model = source["switch_model"];
	        this.protocol = source["protocol"];
	        this.discovered_at = this.convertValues(source["discovered_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SystemInfo {
	    hostname: string;
	    model: string;
	    cpu: string;
	    ram_total_gb: number;
	    os: string;
	    ip_address: string;
	    mac_address: string;
	    gateway: string;
	    network_adapter: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hostname = source["hostname"];
	        this.model = source["model"];
	        this.cpu = source["cpu"];
	        this.ram_total_gb = source["ram_total_gb"];
	        this.os = source["os"];
	        this.ip_address = source["ip_address"];
	        this.mac_address = source["mac_address"];
	        this.gateway = source["gateway"];
	        this.network_adapter = source["network_adapter"];
	    }
	}
	export class LocalMetricRecord {
	    id: number;
	    node_id: string;
	    location: string;
	    group_name?: string;
	    vlan?: number;
	    // Go type: time
	    timestamp: any;
	    download_mbps: number;
	    upload_mbps: number;
	    ping_gateway: number;
	    synced: boolean;
	    error_message?: string;
	    system_info?: SystemInfo;
	    switch_info?: SwitchInfo;
	    modular_results?: ModularResults;
	
	    static createFrom(source: any = {}) {
	        return new LocalMetricRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.node_id = source["node_id"];
	        this.location = source["location"];
	        this.group_name = source["group_name"];
	        this.vlan = source["vlan"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.download_mbps = source["download_mbps"];
	        this.upload_mbps = source["upload_mbps"];
	        this.ping_gateway = source["ping_gateway"];
	        this.synced = source["synced"];
	        this.error_message = source["error_message"];
	        this.system_info = this.convertValues(source["system_info"], SystemInfo);
	        this.switch_info = this.convertValues(source["switch_info"], SwitchInfo);
	        this.modular_results = this.convertValues(source["modular_results"], ModularResults);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class ProbeStatus {
	    is_running: boolean;
	    status_text: string;
	    group_name?: string;
	    current_vlan?: number;
	    // Go type: time
	    last_run?: any;
	    // Go type: time
	    next_run?: any;
	    last_metric?: LocalMetricRecord;
	    server_status: string;
	    total_synced: number;
	    total_pending: number;
	    current_pc?: SystemInfo;
	    current_switch?: SwitchInfo;
	    sync_with_central: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProbeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_running = source["is_running"];
	        this.status_text = source["status_text"];
	        this.group_name = source["group_name"];
	        this.current_vlan = source["current_vlan"];
	        this.last_run = this.convertValues(source["last_run"], null);
	        this.next_run = this.convertValues(source["next_run"], null);
	        this.last_metric = this.convertValues(source["last_metric"], LocalMetricRecord);
	        this.server_status = source["server_status"];
	        this.total_synced = source["total_synced"];
	        this.total_pending = source["total_pending"];
	        this.current_pc = this.convertValues(source["current_pc"], SystemInfo);
	        this.current_switch = this.convertValues(source["current_switch"], SwitchInfo);
	        this.sync_with_central = source["sync_with_central"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	

}

