export namespace main {
	
	export class PortInfo {
	    name: string;
	    vid: string;
	    pid: string;
	    serial: string;
	    product: string;
	    isTinySA: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PortInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.vid = source["vid"];
	        this.pid = source["pid"];
	        this.serial = source["serial"];
	        this.product = source["product"];
	        this.isTinySA = source["isTinySA"];
	    }
	}
	export class Status {
	    connected: boolean;
	    connecting: boolean;
	    port: string;
	    usbDir: string;
	    sweeping: boolean;
	    model: tinysa.Model;
	    battery: string;
	    config?: tinysa.SweepConfig;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.connecting = source["connecting"];
	        this.port = source["port"];
	        this.usbDir = source["usbDir"];
	        this.sweeping = source["sweeping"];
	        this.model = this.convertValues(source["model"], tinysa.Model);
	        this.battery = source["battery"];
	        this.config = this.convertValues(source["config"], tinysa.SweepConfig);
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
	export class WebInfo {
	    running: boolean;
	    port: number;
	    urls: string[];
	    started: string;

	    static createFrom(source: any = {}) {
	        return new WebInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.port = source["port"];
	        this.urls = source["urls"];
	        this.started = source["started"];
	    }
	}

}

export namespace tinysa {
	
	export class Model {
	    ultra: boolean;
	    scale: number;
	    maxFreq: number;
	    firmware: string;
	
	    static createFrom(source: any = {}) {
	        return new Model(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ultra = source["ultra"];
	        this.scale = source["scale"];
	        this.maxFreq = source["maxFreq"];
	        this.firmware = source["firmware"];
	    }
	}
	export class SweepConfig {
	    start: number;
	    stop: number;
	    points: number;
	    rbw: number;
	    atten: number;
	    lna: boolean;
	    spur: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SweepConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.stop = source["stop"];
	        this.points = source["points"];
	        this.rbw = source["rbw"];
	        this.atten = source["atten"];
	        this.lna = source["lna"];
	        this.spur = source["spur"];
	    }
	}

}

