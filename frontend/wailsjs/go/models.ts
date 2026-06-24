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

