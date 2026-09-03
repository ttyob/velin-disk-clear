export namespace agent {
	
	export class ChatMessage {
	    role: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	    }
	}
	export class Finding {
	    item_id: string;
	    rule_id: string;
	    name: string;
	    path: string;
	    directory: string;
	    extension: string;
	    category: string;
	    purpose: string;
	    clean_effect: string;
	    recommendation: string;
	    recommendation_reason: string;
	    risk: string;
	    default_selected: boolean;
	    selectable: boolean;
	    action: string;
	    logical_size: number;
	    allocated_size: number;
	    modified_at: string;
	    help_summary: string;
	    help_details?: string;
	    special_warning?: string;
	    manual_steps?: string[];
	    confidence: number;
	    reason: string;
	    suggested_action: string;
	
	    static createFrom(source: any = {}) {
	        return new Finding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_id = source["item_id"];
	        this.rule_id = source["rule_id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.directory = source["directory"];
	        this.extension = source["extension"];
	        this.category = source["category"];
	        this.purpose = source["purpose"];
	        this.clean_effect = source["clean_effect"];
	        this.recommendation = source["recommendation"];
	        this.recommendation_reason = source["recommendation_reason"];
	        this.risk = source["risk"];
	        this.default_selected = source["default_selected"];
	        this.selectable = source["selectable"];
	        this.action = source["action"];
	        this.logical_size = source["logical_size"];
	        this.allocated_size = source["allocated_size"];
	        this.modified_at = source["modified_at"];
	        this.help_summary = source["help_summary"];
	        this.help_details = source["help_details"];
	        this.special_warning = source["special_warning"];
	        this.manual_steps = source["manual_steps"];
	        this.confidence = source["confidence"];
	        this.reason = source["reason"];
	        this.suggested_action = source["suggested_action"];
	    }
	}
	export class Preset {
	    id: string;
	    name: string;
	    objective: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.objective = source["objective"];
	        this.description = source["description"];
	    }
	}
	export class Request {
	    objective: string;
	    scan_id: string;
	    mode: string;
	    scan_mode: string;
	    roots?: string[];
	    rule_ids?: string[];
	    session_id?: string;
	    messages?: ChatMessage[];
	
	    static createFrom(source: any = {}) {
	        return new Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.objective = source["objective"];
	        this.scan_id = source["scan_id"];
	        this.mode = source["mode"];
	        this.scan_mode = source["scan_mode"];
	        this.roots = source["roots"];
	        this.rule_ids = source["rule_ids"];
	        this.session_id = source["session_id"];
	        this.messages = this.convertValues(source["messages"], ChatMessage);
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
	export class Suggestion {
	    item_id: string;
	    classification: string;
	    recommendation: string;
	    risk: string;
	    confidence: number;
	    reason: string;
	    suggested_action: string;
	
	    static createFrom(source: any = {}) {
	        return new Suggestion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_id = source["item_id"];
	        this.classification = source["classification"];
	        this.recommendation = source["recommendation"];
	        this.risk = source["risk"];
	        this.confidence = source["confidence"];
	        this.reason = source["reason"];
	        this.suggested_action = source["suggested_action"];
	    }
	}
	export class Result {
	    mode: string;
	    summary: string;
	    reply?: string;
	    items: Finding[];
	    suggestions?: Suggestion[];
	    scan_id: string;
	    session_id?: string;
	    provider_model: string;
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.summary = source["summary"];
	        this.reply = source["reply"];
	        this.items = this.convertValues(source["items"], Finding);
	        this.suggestions = this.convertValues(source["suggestions"], Suggestion);
	        this.scan_id = source["scan_id"];
	        this.session_id = source["session_id"];
	        this.provider_model = source["provider_model"];
	        this.truncated = source["truncated"];
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

export namespace cleaner {
	
	export class BuildRequest {
	    scan_id: string;
	    item_ids: string[];
	
	    static createFrom(source: any = {}) {
	        return new BuildRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scan_id = source["scan_id"];
	        this.item_ids = source["item_ids"];
	    }
	}
	export class ExecuteRequest {
	    plan_id: string;
	    confirmation_token: string;
	
	    static createFrom(source: any = {}) {
	        return new ExecuteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plan_id = source["plan_id"];
	        this.confirmation_token = source["confirmation_token"];
	    }
	}
	export class ItemResult {
	    item_id: string;
	    path: string;
	    action: string;
	    status: string;
	    bytes_processed: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ItemResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_id = source["item_id"];
	        this.path = source["path"];
	        this.action = source["action"];
	        this.status = source["status"];
	        this.bytes_processed = source["bytes_processed"];
	        this.error = source["error"];
	    }
	}
	export class PlanItem {
	    id: string;
	    rule_id: string;
	    name: string;
	    path: string;
	    risk: string;
	    action: string;
	    recovery_type: string;
	    allocated_size: number;
	    estimated_reclaimable: number;
	    default_selected: boolean;
	    requires_admin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PlanItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rule_id = source["rule_id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.risk = source["risk"];
	        this.action = source["action"];
	        this.recovery_type = source["recovery_type"];
	        this.allocated_size = source["allocated_size"];
	        this.estimated_reclaimable = source["estimated_reclaimable"];
	        this.default_selected = source["default_selected"];
	        this.requires_admin = source["requires_admin"];
	    }
	}
	export class Plan {
	    id: string;
	    scan_id: string;
	    status: string;
	    items: PlanItem[];
	    item_count: number;
	    default_selected_count: number;
	    manual_selected_count: number;
	    low_risk_count: number;
	    medium_risk_count: number;
	    high_risk_count: number;
	    permanent_delete_count: number;
	    estimated_reclaimable: number;
	    confirmation_token: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    expires_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Plan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.scan_id = source["scan_id"];
	        this.status = source["status"];
	        this.items = this.convertValues(source["items"], PlanItem);
	        this.item_count = source["item_count"];
	        this.default_selected_count = source["default_selected_count"];
	        this.manual_selected_count = source["manual_selected_count"];
	        this.low_risk_count = source["low_risk_count"];
	        this.medium_risk_count = source["medium_risk_count"];
	        this.high_risk_count = source["high_risk_count"];
	        this.permanent_delete_count = source["permanent_delete_count"];
	        this.estimated_reclaimable = source["estimated_reclaimable"];
	        this.confirmation_token = source["confirmation_token"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.expires_at = this.convertValues(source["expires_at"], null);
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
	
	export class Result {
	    id: string;
	    plan_id: string;
	    status: string;
	    succeeded: number;
	    skipped: number;
	    failed: number;
	    deleted_bytes: number;
	    actually_reclaimed: number;
	    items: ItemResult[];
	    // Go type: time
	    started_at: any;
	    // Go type: time
	    completed_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.plan_id = source["plan_id"];
	        this.status = source["status"];
	        this.succeeded = source["succeeded"];
	        this.skipped = source["skipped"];
	        this.failed = source["failed"];
	        this.deleted_bytes = source["deleted_bytes"];
	        this.actually_reclaimed = source["actually_reclaimed"];
	        this.items = this.convertValues(source["items"], ItemResult);
	        this.started_at = this.convertValues(source["started_at"], null);
	        this.completed_at = this.convertValues(source["completed_at"], null);
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

export namespace disks {
	
	export class Volume {
	    id: string;
	    name: string;
	    mount_point: string;
	    file_system: string;
	    total_bytes: number;
	    free_bytes: number;
	    used_bytes: number;
	    system: boolean;
	    ready: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Volume(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.mount_point = source["mount_point"];
	        this.file_system = source["file_system"];
	        this.total_bytes = source["total_bytes"];
	        this.free_bytes = source["free_bytes"];
	        this.used_bytes = source["used_bytes"];
	        this.system = source["system"];
	        this.ready = source["ready"];
	    }
	}

}

export namespace main {
	
	export class Dashboard {
	    volumes: disks.Volume[];
	    rule_count: number;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new Dashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.volumes = this.convertValues(source["volumes"], disks.Volume);
	        this.rule_count = source["rule_count"];
	        this.version = source["version"];
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

export namespace provider {
	
	export class Config {
	    id: string;
	    name: string;
	    protocol: string;
	    base_url: string;
	    model: string;
	    credential_ref: string;
	    key_saved: boolean;
	    timeout_seconds: number;
	    max_output_tokens: number;
	    enabled: boolean;
	    capability_ok: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.base_url = source["base_url"];
	        this.model = source["model"];
	        this.credential_ref = source["credential_ref"];
	        this.key_saved = source["key_saved"];
	        this.timeout_seconds = source["timeout_seconds"];
	        this.max_output_tokens = source["max_output_tokens"];
	        this.enabled = source["enabled"];
	        this.capability_ok = source["capability_ok"];
	    }
	}
	export class ConfigInput {
	    name: string;
	    base_url: string;
	    model: string;
	    api_key: string;
	    timeout_seconds: number;
	    max_output_tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new ConfigInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.base_url = source["base_url"];
	        this.model = source["model"];
	        this.api_key = source["api_key"];
	        this.timeout_seconds = source["timeout_seconds"];
	        this.max_output_tokens = source["max_output_tokens"];
	    }
	}
	export class TestResult {
	    ok: boolean;
	    message: string;
	    model_found: boolean;
	    models: string[];
	    endpoint: string;
	    http_status: number;
	    capability_ok: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.model_found = source["model_found"];
	        this.models = source["models"];
	        this.endpoint = source["endpoint"];
	        this.http_status = source["http_status"];
	        this.capability_ok = source["capability_ok"];
	    }
	}

}

export namespace rules {
	
	export class ActionSpec {
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	    }
	}
	export class Help {
	    summary: string;
	    details?: string;
	    special_warning?: string;
	    manual_steps?: string[];
	    learn_more_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new Help(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = source["summary"];
	        this.details = source["details"];
	        this.special_warning = source["special_warning"];
	        this.manual_steps = source["manual_steps"];
	        this.learn_more_url = source["learn_more_url"];
	    }
	}
	export class SafetySpec {
	    allowed_roots: string[];
	    revalidate_before_clean: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SafetySpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowed_roots = source["allowed_roots"];
	        this.revalidate_before_clean = source["revalidate_before_clean"];
	    }
	}
	export class ScanSpec {
	    roots: string[];
	    include?: string[];
	    exclude?: string[];
	    extensions?: string[];
	    min_age?: string;
	    min_size_bytes?: number;
	    stay_on_volume: boolean;
	    follow_reparse_points: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScanSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.roots = source["roots"];
	        this.include = source["include"];
	        this.exclude = source["exclude"];
	        this.extensions = source["extensions"];
	        this.min_age = source["min_age"];
	        this.min_size_bytes = source["min_size_bytes"];
	        this.stay_on_volume = source["stay_on_volume"];
	        this.follow_reparse_points = source["follow_reparse_points"];
	    }
	}
	export class Rule {
	    id: string;
	    version: number;
	    name: string;
	    description: string;
	    purpose: string;
	    clean_effect: string;
	    recommendation: string;
	    recommendation_reason: string;
	    category: string;
	    platform: string;
	    enabled: boolean;
	    risk: string;
	    default_selected: boolean;
	    requires_admin: boolean;
	    supported_windows_versions: string[];
	    scope: string;
	    size_mode: string;
	    recovery_type: string;
	    requires_network_after_clean: boolean;
	    may_sign_out: boolean;
	    requires_restart: boolean;
	    process_guard: string[];
	    conflicts: string[];
	    last_verified_at: string;
	    source: string;
	    modes: string[];
	    help: Help;
	    scan: ScanSpec;
	    action: ActionSpec;
	    safety: SafetySpec;
	
	    static createFrom(source: any = {}) {
	        return new Rule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.version = source["version"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.purpose = source["purpose"];
	        this.clean_effect = source["clean_effect"];
	        this.recommendation = source["recommendation"];
	        this.recommendation_reason = source["recommendation_reason"];
	        this.category = source["category"];
	        this.platform = source["platform"];
	        this.enabled = source["enabled"];
	        this.risk = source["risk"];
	        this.default_selected = source["default_selected"];
	        this.requires_admin = source["requires_admin"];
	        this.supported_windows_versions = source["supported_windows_versions"];
	        this.scope = source["scope"];
	        this.size_mode = source["size_mode"];
	        this.recovery_type = source["recovery_type"];
	        this.requires_network_after_clean = source["requires_network_after_clean"];
	        this.may_sign_out = source["may_sign_out"];
	        this.requires_restart = source["requires_restart"];
	        this.process_guard = source["process_guard"];
	        this.conflicts = source["conflicts"];
	        this.last_verified_at = source["last_verified_at"];
	        this.source = source["source"];
	        this.modes = source["modes"];
	        this.help = this.convertValues(source["help"], Help);
	        this.scan = this.convertValues(source["scan"], ScanSpec);
	        this.action = this.convertValues(source["action"], ActionSpec);
	        this.safety = this.convertValues(source["safety"], SafetySpec);
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

export namespace scanner {
	
	export class ErrorItem {
	    path: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ErrorItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.message = source["message"];
	    }
	}
	export class Folder {
	    id: string;
	    name: string;
	    path: string;
	    file_count: number;
	    logical_bytes: number;
	    allocated_bytes: number;
	    estimated_reclaimable: number;
	    highest_risk: string;
	    children: Folder[];
	    item_ids: string[];
	
	    static createFrom(source: any = {}) {
	        return new Folder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.file_count = source["file_count"];
	        this.logical_bytes = source["logical_bytes"];
	        this.allocated_bytes = source["allocated_bytes"];
	        this.estimated_reclaimable = source["estimated_reclaimable"];
	        this.highest_risk = source["highest_risk"];
	        this.children = this.convertValues(source["children"], Folder);
	        this.item_ids = source["item_ids"];
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
	export class Item {
	    id: string;
	    rule_id: string;
	    matched_rule_ids: string[];
	    name: string;
	    path: string;
	    directory: string;
	    extension: string;
	    category: string;
	    purpose: string;
	    clean_effect: string;
	    recommendation: string;
	    recommendation_reason: string;
	    risk: string;
	    default_selected: boolean;
	    selectable: boolean;
	    action: string;
	    recovery_type: string;
	    requires_admin: boolean;
	    requires_restart: boolean;
	    logical_size: number;
	    allocated_size: number;
	    estimated_reclaimable: number;
	    volume_id: string;
	    file_id: string;
	    link_count: number;
	    // Go type: time
	    modified_at: any;
	    help_summary: string;
	    help_details?: string;
	    special_warning?: string;
	    manual_steps?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rule_id = source["rule_id"];
	        this.matched_rule_ids = source["matched_rule_ids"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.directory = source["directory"];
	        this.extension = source["extension"];
	        this.category = source["category"];
	        this.purpose = source["purpose"];
	        this.clean_effect = source["clean_effect"];
	        this.recommendation = source["recommendation"];
	        this.recommendation_reason = source["recommendation_reason"];
	        this.risk = source["risk"];
	        this.default_selected = source["default_selected"];
	        this.selectable = source["selectable"];
	        this.action = source["action"];
	        this.recovery_type = source["recovery_type"];
	        this.requires_admin = source["requires_admin"];
	        this.requires_restart = source["requires_restart"];
	        this.logical_size = source["logical_size"];
	        this.allocated_size = source["allocated_size"];
	        this.estimated_reclaimable = source["estimated_reclaimable"];
	        this.volume_id = source["volume_id"];
	        this.file_id = source["file_id"];
	        this.link_count = source["link_count"];
	        this.modified_at = this.convertValues(source["modified_at"], null);
	        this.help_summary = source["help_summary"];
	        this.help_details = source["help_details"];
	        this.special_warning = source["special_warning"];
	        this.manual_steps = source["manual_steps"];
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
	export class ItemPage {
	    items: Item[];
	    total: number;
	    offset: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new ItemPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Item);
	        this.total = source["total"];
	        this.offset = source["offset"];
	        this.limit = source["limit"];
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
	export class Job {
	    id: string;
	    status: string;
	    mode: string;
	    current_path: string;
	    scanned_files: number;
	    matched_files: number;
	    logical_bytes: number;
	    allocated_bytes: number;
	    estimated_reclaimable: number;
	    error_count: number;
	    errors?: ErrorItem[];
	    // Go type: time
	    started_at: any;
	    // Go type: time
	    completed_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.mode = source["mode"];
	        this.current_path = source["current_path"];
	        this.scanned_files = source["scanned_files"];
	        this.matched_files = source["matched_files"];
	        this.logical_bytes = source["logical_bytes"];
	        this.allocated_bytes = source["allocated_bytes"];
	        this.estimated_reclaimable = source["estimated_reclaimable"];
	        this.error_count = source["error_count"];
	        this.errors = this.convertValues(source["errors"], ErrorItem);
	        this.started_at = this.convertValues(source["started_at"], null);
	        this.completed_at = this.convertValues(source["completed_at"], null);
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
	export class Request {
	    mode: string;
	    roots: string[];
	    rule_ids: string[];
	
	    static createFrom(source: any = {}) {
	        return new Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.roots = source["roots"];
	        this.rule_ids = source["rule_ids"];
	    }
	}

}

