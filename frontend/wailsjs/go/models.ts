export namespace models {
	
	export class ApiNode {
	    id: string;
	    type: string;
	    name: string;
	    method?: string;
	    requestType?: string;
	    path: string;
	    url?: string;
	    children?: ApiNode[];
	
	    static createFrom(source: any = {}) {
	        return new ApiNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.method = source["method"];
	        this.requestType = source["requestType"];
	        this.path = source["path"];
	        this.url = source["url"];
	        this.children = this.convertValues(source["children"], ApiNode);
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
	export class Variable {
	    key: string;
	    value: string;
	    enabled: boolean;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new Variable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.enabled = source["enabled"];
	        this.description = source["description"];
	    }
	}
	export class Header {
	    key: string;
	    value: string;
	    enabled: boolean;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new Header(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.enabled = source["enabled"];
	        this.description = source["description"];
	    }
	}
	export class DirMeta {
	    id: string;
	    name: string;
	    headers: Header[];
	    variables: Variable[];
	
	    static createFrom(source: any = {}) {
	        return new DirMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.headers = this.convertValues(source["headers"], Header);
	        this.variables = this.convertValues(source["variables"], Variable);
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
	export class QueryParam {
	    key: string;
	    value: string;
	    enabled: boolean;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new QueryParam(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.enabled = source["enabled"];
	        this.description = source["description"];
	    }
	}
	export class FetchParseResult {
	    url: string;
	    method: string;
	    headers: Record<string, string>;
	    query: QueryParam[];
	    body: string;
	
	    static createFrom(source: any = {}) {
	        return new FetchParseResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.method = source["method"];
	        this.headers = source["headers"];
	        this.query = this.convertValues(source["query"], QueryParam);
	        this.body = source["body"];
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
	export class FormField {
	    key: string;
	    value: string;
	    type: string;
	    enabled: boolean;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new FormField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.type = source["type"];
	        this.enabled = source["enabled"];
	        this.description = source["description"];
	    }
	}
	
	export class ScriptResult {
	    success: boolean;
	    error?: string;
	    durationMs: number;
	    logs: string[];
	
	    static createFrom(source: any = {}) {
	        return new ScriptResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.durationMs = source["durationMs"];
	        this.logs = source["logs"];
	    }
	}
	export class HttpResponse {
	    statusCode: number;
	    status: string;
	    headers: Record<string, Array<string>>;
	    cookies: string[];
	    body: string;
	    timeMs: number;
	    serverTimeMs: number;
	    transferTimeMs: number;
	    sizeBytes: number;
	    contentType: string;
	    preScript?: ScriptResult;
	    postScript?: ScriptResult;
	
	    static createFrom(source: any = {}) {
	        return new HttpResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.statusCode = source["statusCode"];
	        this.status = source["status"];
	        this.headers = source["headers"];
	        this.cookies = source["cookies"];
	        this.body = source["body"];
	        this.timeMs = source["timeMs"];
	        this.serverTimeMs = source["serverTimeMs"];
	        this.transferTimeMs = source["transferTimeMs"];
	        this.sizeBytes = source["sizeBytes"];
	        this.contentType = source["contentType"];
	        this.preScript = this.convertValues(source["preScript"], ScriptResult);
	        this.postScript = this.convertValues(source["postScript"], ScriptResult);
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
	export class LoadTestConfig {
	    concurrency: number;
	    mode: string;
	    rounds: number;
	    durationSeconds: number;
	    logEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LoadTestConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.concurrency = source["concurrency"];
	        this.mode = source["mode"];
	        this.rounds = source["rounds"];
	        this.durationSeconds = source["durationSeconds"];
	        this.logEnabled = source["logEnabled"];
	    }
	}
	
	export class Request {
	    id: string;
	    type: string;
	    name: string;
	    method: string;
	    url: string;
	    headers: Header[];
	    query: QueryParam[];
	    bodyType: string;
	    body: string;
	    formData: FormField[];
	    preRequestScript: string;
	    postResponseScript: string;
	    description: string;
	    disabledInheritedHeaderKeys: string[];
	
	    static createFrom(source: any = {}) {
	        return new Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.headers = this.convertValues(source["headers"], Header);
	        this.query = this.convertValues(source["query"], QueryParam);
	        this.bodyType = source["bodyType"];
	        this.body = source["body"];
	        this.formData = this.convertValues(source["formData"], FormField);
	        this.preRequestScript = source["preRequestScript"];
	        this.postResponseScript = source["postResponseScript"];
	        this.description = source["description"];
	        this.disabledInheritedHeaderKeys = source["disabledInheritedHeaderKeys"];
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
	export class RequestPreview {
	    url: string;
	    headers: Header[];
	    query: QueryParam[];
	    bodyType: string;
	    body: string;
	    formData: FormField[];
	    contentType: string;
	
	    static createFrom(source: any = {}) {
	        return new RequestPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.headers = this.convertValues(source["headers"], Header);
	        this.query = this.convertValues(source["query"], QueryParam);
	        this.bodyType = source["bodyType"];
	        this.body = source["body"];
	        this.formData = this.convertValues(source["formData"], FormField);
	        this.contentType = source["contentType"];
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
	
	export class Settings {
	    version: string;
	    theme: string;
	    requestTimeoutMs: number;
	    globalHeaders: Header[];
	    globalVariables: Variable[];
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.theme = source["theme"];
	        this.requestTimeoutMs = source["requestTimeoutMs"];
	        this.globalHeaders = this.convertValues(source["globalHeaders"], Header);
	        this.globalVariables = this.convertValues(source["globalVariables"], Variable);
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
	
	export class WorkspaceMigrationResult {
	    sourcePath: string;
	    targetPath: string;
	    copiedFiles: number;
	    skippedFiles: number;
	    createdDirs: number;
	    failedFiles: number;
	    failureSamples: string[];
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceMigrationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourcePath = source["sourcePath"];
	        this.targetPath = source["targetPath"];
	        this.copiedFiles = source["copiedFiles"];
	        this.skippedFiles = source["skippedFiles"];
	        this.createdDirs = source["createdDirs"];
	        this.failedFiles = source["failedFiles"];
	        this.failureSamples = source["failureSamples"];
	        this.message = source["message"];
	    }
	}

}

