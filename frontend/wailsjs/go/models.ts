export namespace acp {
	
	export class Annotations {
	    audience?: string[];
	    priority?: number;
	    lastModified?: string;
	
	    static createFrom(source: any = {}) {
	        return new Annotations(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.audience = source["audience"];
	        this.priority = source["priority"];
	        this.lastModified = source["lastModified"];
	    }
	}
	export class ContentBlock {
	    type: string;
	    text?: string;
	    data?: string;
	    mimeType?: string;
	    uri?: string;
	    name?: string;
	    description?: string;
	    size?: number;
	    title?: string;
	    resource?: number[];
	    annotations?: Annotations;
	
	    static createFrom(source: any = {}) {
	        return new ContentBlock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.text = source["text"];
	        this.data = source["data"];
	        this.mimeType = source["mimeType"];
	        this.uri = source["uri"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.size = source["size"];
	        this.title = source["title"];
	        this.resource = source["resource"];
	        this.annotations = this.convertValues(source["annotations"], Annotations);
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
	export class Cost {
	    amount: number;
	    currency: string;
	
	    static createFrom(source: any = {}) {
	        return new Cost(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.amount = source["amount"];
	        this.currency = source["currency"];
	    }
	}
	export class CurrentModeUpdate {
	    currentModeId: string;
	
	    static createFrom(source: any = {}) {
	        return new CurrentModeUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentModeId = source["currentModeId"];
	    }
	}
	export class PlanEntry {
	    content: string;
	    status: string;
	    priority: string;
	
	    static createFrom(source: any = {}) {
	        return new PlanEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	    }
	}
	export class Plan {
	    entries: PlanEntry[];
	
	    static createFrom(source: any = {}) {
	        return new Plan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entries = this.convertValues(source["entries"], PlanEntry);
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
	
	export class ToolCallLocation {
	    path: string;
	    line?: number;
	
	    static createFrom(source: any = {}) {
	        return new ToolCallLocation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.line = source["line"];
	    }
	}
	export class ToolCallContent {
	    type: string;
	    content?: ContentBlock;
	    path?: string;
	    oldText?: string;
	    newText?: string;
	    terminalId?: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolCallContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.content = this.convertValues(source["content"], ContentBlock);
	        this.path = source["path"];
	        this.oldText = source["oldText"];
	        this.newText = source["newText"];
	        this.terminalId = source["terminalId"];
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
	export class ToolCall {
	    toolCallId: string;
	    title: string;
	    kind: string;
	    status: string;
	    content: ToolCallContent[];
	    locations: ToolCallLocation[];
	    rawInput: number[];
	    rawOutput: number[];
	
	    static createFrom(source: any = {}) {
	        return new ToolCall(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolCallId = source["toolCallId"];
	        this.title = source["title"];
	        this.kind = source["kind"];
	        this.status = source["status"];
	        this.content = this.convertValues(source["content"], ToolCallContent);
	        this.locations = this.convertValues(source["locations"], ToolCallLocation);
	        this.rawInput = source["rawInput"];
	        this.rawOutput = source["rawOutput"];
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
	
	
	export class UsageUpdate {
	    size: number;
	    used: number;
	    cost?: Cost;
	
	    static createFrom(source: any = {}) {
	        return new UsageUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.size = source["size"];
	        this.used = source["used"];
	        this.cost = this.convertValues(source["cost"], Cost);
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

export namespace agent {
	
	export class AgentInfo {
	    name: string;
	    version: string;
	    loadSession: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.loadSession = source["loadSession"];
	    }
	}
	export class Session {
	    id: string;
	    cwd: string;
	    title?: string;
	    updatedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.cwd = source["cwd"];
	        this.title = source["title"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}

export namespace channels {
	
	export class Status {
	    running: boolean;
	    command?: string;
	    pid?: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.command = source["command"];
	        this.pid = source["pid"];
	        this.error = source["error"];
	    }
	}

}

export namespace config {
	
	export class ACPAgentConfig {
	    id: string;
	    name?: string;
	    enabled?: boolean;
	    type?: string;
	    command?: string;
	    args?: string[];
	    url?: string;
	    headers?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new ACPAgentConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.type = source["type"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.url = source["url"];
	        this.headers = source["headers"];
	    }
	}
	export class LoggingConfig {
	    enabled?: boolean;
	    level?: string;
	    stdout?: boolean;
	    file?: string;
	
	    static createFrom(source: any = {}) {
	        return new LoggingConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.level = source["level"];
	        this.stdout = source["stdout"];
	        this.file = source["file"];
	    }
	}
	export class ProviderConfig {
	    apiKey: string;
	    apiBase?: string;
	    type?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProviderConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKey = source["apiKey"];
	        this.apiBase = source["apiBase"];
	        this.type = source["type"];
	    }
	}
	export class ProfileConfig {
	    provider: string;
	    model?: string;
	    workspace?: string;
	    maxTokens?: number;
	    temperature?: number;
	    contextWindowTokens?: number;
	    contextWarnRatio?: number;
	
	    static createFrom(source: any = {}) {
	        return new ProfileConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.workspace = source["workspace"];
	        this.maxTokens = source["maxTokens"];
	        this.temperature = source["temperature"];
	        this.contextWindowTokens = source["contextWindowTokens"];
	        this.contextWarnRatio = source["contextWarnRatio"];
	    }
	}
	export class Config {
	    agents?: ACPAgentConfig[];
	    profiles: Record<string, ProfileConfig>;
	    providers: Record<string, ProviderConfig>;
	    mcpServers?: Record<string, mcp.McpServerConfig>;
	    channels?: Record<string, any>;
	    tools?: Record<string, Array<number>>;
	    logging?: LoggingConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agents = this.convertValues(source["agents"], ACPAgentConfig);
	        this.profiles = this.convertValues(source["profiles"], ProfileConfig, true);
	        this.providers = this.convertValues(source["providers"], ProviderConfig, true);
	        this.mcpServers = this.convertValues(source["mcpServers"], mcp.McpServerConfig, true);
	        this.channels = source["channels"];
	        this.tools = source["tools"];
	        this.logging = this.convertValues(source["logging"], LoggingConfig);
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

export namespace conversation {
	
	export class Block {
	    id: string;
	    type: string;
	    content?: string;
	    data?: string;
	    mime?: string;
	    tool?: acp.ToolCall;
	    plan?: acp.Plan;
	    raw?: number[];
	
	    static createFrom(source: any = {}) {
	        return new Block(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.content = source["content"];
	        this.data = source["data"];
	        this.mime = source["mime"];
	        this.tool = this.convertValues(source["tool"], acp.ToolCall);
	        this.plan = this.convertValues(source["plan"], acp.Plan);
	        this.raw = source["raw"];
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
	export class Message {
	    id: string;
	    conversationId: string;
	    role: string;
	    protocolMessageId?: string;
	    blocks: Block[];
	    status: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.conversationId = source["conversationId"];
	        this.role = source["role"];
	        this.protocolMessageId = source["protocolMessageId"];
	        this.blocks = this.convertValues(source["blocks"], Block);
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class Source {
	    type: string;
	    channel?: string;
	    accountId?: string;
	    threadId?: string;
	
	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.channel = source["channel"];
	        this.accountId = source["accountId"];
	        this.threadId = source["threadId"];
	    }
	}
	export class Conversation {
	    id: string;
	    acpSessionId: string;
	    runtimeId?: string;
	    title?: string;
	    cwd?: string;
	    source: Source;
	    messages: Message[];
	    usage?: acp.UsageUpdate;
	    mode?: acp.CurrentModeUpdate;
	    updatedAt: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Conversation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.acpSessionId = source["acpSessionId"];
	        this.runtimeId = source["runtimeId"];
	        this.title = source["title"];
	        this.cwd = source["cwd"];
	        this.source = this.convertValues(source["source"], Source);
	        this.messages = this.convertValues(source["messages"], Message);
	        this.usage = this.convertValues(source["usage"], acp.UsageUpdate);
	        this.mode = this.convertValues(source["mode"], acp.CurrentModeUpdate);
	        this.updatedAt = source["updatedAt"];
	        this.createdAt = source["createdAt"];
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

export namespace mcp {
	
	export class McpServerConfig {
	    type?: string;
	    command?: string;
	    args?: string[];
	    env?: Record<string, string>;
	    url?: string;
	    headers?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new McpServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.env = source["env"];
	        this.url = source["url"];
	        this.headers = source["headers"];
	    }
	}

}

