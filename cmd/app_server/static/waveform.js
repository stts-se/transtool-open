"use strict";

// using http://wavesurfer-js.org with modules: regions, timeline

const
deleteKeyCode = 46,
enterKeyCode = 13,
spaceKeyCode = 32,
rightArrowKeyCode = 39,
leftArrowKeyCode = 37,
endKeyCode = 35,
homeKeyCode = 36
;

class Waveform {

    constructor(options) {
	this.options = options;
	console.log("waveform constructor called with options", options);

	var wsPlugins = [
	    WaveSurfer.regions.create({
		// dragSelection is for adding regions
		dragSelection: {
		    slop: 5
		}
	    }),
	    // WaveSurfer.cursor.create({
	    // 	//width: '1px',
	    // 	color: 'orange',
	    // 	style: 'solid',
	    // 	followCursorY: false,
	    // 	opacity: '1',
	    // 	//showTime: true,
	    // }),
	];

	if (this.options.timelineElementID)
	    wsPlugins.push(WaveSurfer.timeline.create({
		container: '#' + this.options.timelineElementID
	    }));

	if (this.options.spectrogramElementID)
	    wsPlugins.push(WaveSurfer.spectrogram.create({
		container: '#' + this.options.spectrogramElementID,
		labels: true,
	    }));


	if (!this.autoplayEnabledFunc)
	    this.autoplayEnabledFunc = function () { return false; }

	this.defaultRegionBackground = 'hsla(0, 0%, 75%, 0.3)'; // 'hsla(200, 50%, 70%, 0.4)';
	this.selectedRegionBackground = 'hsla(120, 100%, 75%, 0.3)'; // this.defaultRegionBackground; //
	var wsOptions = {
	    container: '#' + this.options.waveformElementID,
	    waveColor: 'purple',
	    progressColor: 'purple',
	    loaderColor: 'purple',
	    cursorColor: "orange",
	    autoCenter: true,
	    barHeight: 3,
	    height: 200,
	    plugins: wsPlugins,
	    normalise: true,
	};


	this.wavesurfer = WaveSurfer.create(wsOptions);

	// if (this.options.navigationElementID) {
	// 	document.getElementById(this.options.navigationElementID).innerHTML = `<span class='btn noborder' id='waveform-playpause'>&#x23EF;</span>
	// 	| <span class='btn noborder' id='waveform-skiptofirst'>&#x23EE;</span>
	//     <span class='btn noborder' id='waveform-skipback'>&#x23EA;</span>
	//     <span class='btn noborder' id='waveform-skipforward'>&#x23E9;</span>
	//     <span class='btn noborder' id='waveform-skiptolast'>&#x23ED;</span>
	// </span>`;
	// }
	if (this.options.zoomElementID) {
	    document.getElementById(this.options.zoomElementID).innerHTML = `<span class="slidecontainer" style="vertical-align: middle; display:inline">waveform zoom
		<input id="waveform-zoom-input" title="Waveform zoom" style="vertical-align: middle; display:inline" type="range" min="20" max="1000" value="0" class="slider">
	</span>`;
	}

	let main = this;

	this.wavesurfer.on("audioprocess", function (evt) {
	    main.debug("audioprocess", evt);
	});

	this.wavesurfer.on("region-play", function (evt) {
	    console.log("region-play");
	});

	this.wavesurfer.on("pause", function () {
	    console.log("wavesurfer pause");
	});

	this.wavesurfer.on("play", function () {
	    console.log("wavesurfer play");
	});

	this.wavesurfer.on("region-out", function (evt) {
	    //HB 0714 if (/* !main.continuousPlay && */ main.wavesurfer.isPlaying())
	    console.log("region-out - continuousPlay: " + main.continuousPlay);
	    console.log("region-out - isPlaying: " + main.wavesurfer.isPlaying());
	    if (main.continuousPlay !== true && main.wavesurfer.isPlaying()) {
		console.log("region-out - pausing wavesurfer");
	    	main.wavesurfer.pause();
	    }
	});

	//HB 0714 on region-in, used with play-all to display region content in editor
	//HB 0728 Doesn't work right, causes colors to disappear from all chunks when playing
	//With this, "play all" works as expected, but other things don't
	this.wavesurfer.on("region-in", function (region) {
	    console.log("region-in");
	    //console.log(region);
	    //console.log(region.uuid);
	    //#159: last arg should be true to stop pause behaviour.. But only if in "play selected", not in "play all"
	    //I was thinking !main.continuousPlay could work for that - but it seems no.. Yes but also if undefined..
	    console.log("continuousPlay: " + main.continuousPlay);
	    //main.setSelectedRegion(region, false, false);
	    var blockOnSelectedRegionChange = false;
	    if ( main.continuousPlay==null || main.continuousPlay===false ) {
		blockOnSelectedRegionChange = true;
		//HB 210823 Added return here as a workaround to the problem where after "play selected" the next chunk is selected
		//Not perfect but at least the jump is gone
		return;
	    }
	    console.log("blockOnSelectedRegionChange: " + blockOnSelectedRegionChange);
	    main.setSelectedRegion(region, false, blockOnSelectedRegionChange);
	    //console.log("END wavesurfer.on(\"region-in\")");
	});

	this.wavesurfer.on("region-created", async function (region) {
	    region.color = main.defaultRegionBackground;
	    region.element.addEventListener("click", async function (evt) {
		main.debug("click", evt);
		main.logEvent(evt);
		main.setSelectedRegion(region, false);
		if (/*evt.ctrlKey || */ main.autoplayEnabledFunc()) {
		    await LIB.sleep(100);
		    region.play();
		}
	    });

	    //console.log("region-created", "uuid", region.uuid, "id", region.id);

	    if (!LIB.validUUIDv4(region.id)) { // this is a manually created region
		region.id = LIB.uuidv4();
		main.setSelectedRegion(region);
	    }
	    if (!region.uuid) {
		region.uuid = region.id;
	    }
	    
	    console.log("region-created", "id", region.id);

	    region.element.title = main.floatWithDecimals(region.start, 2) + " - " + main.floatWithDecimals(region.end, 2) + " (" + region.id + ")";
	    region.drag = false;

	    // main.setSelectedRegion(region);
	    // if (main.onSelectedRegionChange)
	    // 	main.onSelectedRegionChange(region.uuid);
	});

	//this.wavesurfer.on("region-removed", async function (region) {
	//console.log("region-removed", "id", region.id);
	//region.element.title = main.floatWithDecimals(region.start, 2) + " - " + main.floatWithDecimals(region.end, 2) + " (" + region.id + ")";
	//main.updateRegionsMinMax();
	//});

	this.wavesurfer.on("region-update-end", async function (region) {
	    //console.log("region-update-end", "id", region.id, main.listRegions().length);
	    region.element.title = main.floatWithDecimals(region.start, 2) + " - " + main.floatWithDecimals(region.end, 2) + " (" + region.id + ")";
	    main.updateRegionsMinMax(region);
	    main.onUserAddedRegion(region);
	});

	this.wavesurfer.on("region-updated", async function (region) {
	    //console.log("region-updated", region);
	    if (region.element.classList.contains("selected"))
		main.setSelectedRegion(region, false, true);
	    // region.element.addEventListener("contextmenu", function (evt) {
	    //     console.log("rightclick", evt);
	    //     evt.preventDefault();
	    //     return false;
	    // });
	    region.element.title = main.floatWithDecimals(region.start, 2) + " - " + main.floatWithDecimals(region.end, 2) + " (" + region.id + ")";
	    main.updateRegionsMinMax(region);
	});

	this.wavesurfer.on("error", function (evt) {
	    //throw evt; // TODO
	});

	this.wavesurfer.on("ready", function () {
	    console.log("wavesurfer ready");
	    let wave = document.getElementById(this.options.waveformElementID).getElementsByTagName("wave")[0];
	    wave.style["height"] = "178px";
	});

	if (this.options.zoomElementID) {
	    document.getElementById('waveform-zoom-input').addEventListener("input", function (evt) {
		if (!evt.target.classList.contains("disabled")) {
		    let value = evt.target.value;
		    //let selected = main.getSelectedRegion();
		    main.wavesurfer.zoom(Number(value));
		    // if (selected)
		    // 	main.setSelectedRegion(selected, false, false);
		}
	    });
	}

	// if (this.options.navigationElementID) {
	// document.getElementById("waveform-playpause").addEventListener("click", function (evt) {
	// 	main.logEvent(evt);
	// 	if (main.wavesurfer.isPlaying())
	// 		main.wavesurfer.pause();
	// 	else
	// 		main.wavesurfer.play();
	// });

	// document.getElementById("waveform-skipforward").addEventListener("click", function (evt) {
	// 	main.logEvent(evt);
	// 	main.selectNextRegion();
	// });


	// document.getElementById("waveform-skipback").addEventListener("click", function (evt) {
	// 	main.logEvent(evt);
	// 	main.selectPrevRegion();
	// });

	// document.getElementById("waveform-skiptolast").addEventListener("click", function (evt) {
	// 	main.logEvent(evt);
	// 	let regions = main.listRegions();
	// 	if (regions.length > 0)
	// 		main.setSelectedRegion(regions[regions.length - 1]);
	// });


	// document.getElementById("waveform-skiptofirst").addEventListener("click", function (evt) {
	// 	main.logEvent(evt);
	// 	main.setSelectedIndex(0);
	// });
	// }


	document.addEventListener("keydown", function (evt) {
	    main.logEvent(evt);

	    // deletion moved to app.js
	    // if (evt.ctrlKey && evt.keyCode === deleteKeyCode) {
	    // 	let regions = main.listRegions();
	    // 	for (let id in regions) {
	    // 		let region = regions[id];
	    // 		if (region.element.classList.contains("selected")) {
	    // 			region.remove();
	    // 		}
	    // 	}
	    // }
	    return true;
	    // if (this.options.navigationElementID) {
	    // } else if (evt.keyCode === rightArrowKeyCode && evt.ctrlKey) {
	    // 	document.getElementById("waveform-skipforward").click();
	    // } else if (evt.keyCode === leftArrowKeyCode && evt.ctrlKey) {
	    // 	document.getElementById("waveform-skipback").click();
	    // } else if (evt.keyCode === homeKeyCode && evt.ctrlKey) {
	    // 	document.getElementById("waveform-skiptofirst").click();
	    // } else if (evt.keyCode === endKeyCode && evt.ctrlKey) {
	    // 	document.getElementById("waveform-skiptolast").click();
	    // } else if (evt.ctrlKey && evt.keyCode === spaceKeyCode) {
	    // 	if (main.wavesurfer.isPlaying())
	    // 		main.wavesurfer.pause();
	    // 	else
	    // 		main.wavesurfer.play();
	    // }
	    // return true;
	});

	let waveformResizeObserver = new ResizeObserver(function (source) {
	    let pane = document.getElementById("waveform-pane"); // source[0].target;
	    let waveform = document.getElementById("waveform"); // source[0].target;
	    let wave = waveform.getElementsByTagName("wave")[0];
	    if (waveform.style.height) {
		wave.style.height = waveform.style.height;
	    }
	    if (waveform.style.width && (waveform.offsetWidth + 22) != pane.offsetWidth) {
		pane.style.width = (waveform.offsetWidth + 22) + "px";
	    }
	});
	waveformResizeObserver.observe(document.querySelector("#" + this.options.waveformElementID));

	console.log("waveform ready");
    }

    updateRegionsMinMax(region) {
	if (this.options.regionMinLength)
	    region.minLength = this.options.regionMinLength;
	if (this.options.regionMaxLength)
	    region.maxLength = this.options.regionMaxLength;
	
	// stop user from accidentally overlapping regions
	let minGap = 0.1;
	let regions = this.listRegions();
	let thisI = -1;
	for (let i = 0; i < regions.length; i++) {
	    if (regions[i].id === region.id)
		thisI = i;
	}
	if (thisI < regions.length - 1) {
	    let nextRegion = regions[thisI + 1];
	    //region.maxLength = region.start + (nextRegion.start - minGap);
	    if (nextRegion && region.end >= nextRegion.start) {
		region.end = nextRegion.start - minGap;
		//nextRegion.start = region.end + minGap;
	    }
	}
	if (thisI > 0) {
	    let prevRegion = regions[thisI - 1];
	    if (prevRegion && region.start <= prevRegion.end) {
		region.start = prevRegion.end + minGap;
		//prevRegion.end = region.start - minGap;
	    }
	}


	// let regions = Object.values(this.wavesurfer.regions.list);
	// regions.sort(function (a, b) { return a.start - b.start });
	// for (let i = 0; i < regions.length; i++) {
	// 	let region = regions[i];
	// 	region.minLength = 0.1;
	// 	if (i < regions.length-1) {
	// 		let nextRegion = regions[i];
	// 		region.maxLength = region.start + nextRegion.start;
	// 		//console.log("region.maxLength", region.id, region.maxLength);
	// 	}
	// }
	// let regionsX = Object.values(this.wavesurfer.regions.list);
	// for (let i = 0; i < regionsX.length; i++) {
	// 	let region = regionsX[i];
	// 	//console.log("region.maxLength ???", region.id, region.maxLength);
	// }
    }

    setEnabled(enable) {
	let buttons = [
	    document.getElementById("waveform-playpause"),
	    document.getElementById("waveform-skipforward"),
	    document.getElementById("waveform-skipback"),
	    document.getElementById("waveform-skiptofirst"),
	    document.getElementById("waveform-skiptolast"),
	    document.getElementById("waveform-zoom-input"),
	];
	if (enable) {
	    for (let i = 0; i < buttons.length; i++) {
		let btn = buttons[i];
		if (btn) {
		    btn.classList.remove("disabled");
		    btn.removeAttribute("disabled");
		    btn.disabled = false;
		}
	    }
	} else {
	    for (let i = 0; i < buttons.length; i++) {
		let btn = buttons[i];
		if (btn) {
		    btn.classList.add("disabled");
		    btn.disabled = true;
		}
	    }
	}
    }

    play(start, end) {
	//this.continuousPlay = true;
	this.wavesurfer.play(start, end);
    }

    playRegionWithUUID(uuid) {
	let regions = this.listRegions();
	for (let i = 0; i < regions.length; i++) {
	    if (regions[i].uuid === uuid) {
		//console.log("playRegionUUID debug", uuid, regions[i]);
		this.playRegion(regions[i]);
		break;
	    }
	}
    }

    playRegionIndex(index) {
	let regions = this.listRegions();
	for (let i = 0; i < regions.length; i++) {
	    if (i === index) {
		//console.log("playRegionIndex debug", i, index, regions[i]);
		this.playRegion(regions[i]);
		break;
	    }
	}
    }

    playLeftOfRegionIndex(index) {
	let regions = this.listRegions();
	for (let i = 0; i < regions.length; i++) {
	    if (i === index) {
		this.wavesurfer.play(0, regions[i].start);
		break;
	    }
	}
    }

    playRightOfRegionIndex(index) {
	let regions = this.listRegions();
	for (let i = 0; i < regions.length; i++) {
	    if (i === index) {
		this.wavesurfer.play(regions[i].end);
		break;
	    }
	}
    }

    async loadAudioBlob(blob, timeChunks) {
	//console.log("waveform loadAudioBlob", timeChunks);
	await this.wavesurfer.regions.clear();
	await this.wavesurfer.loadBlob(blob);
	await this.loadChunks(timeChunks, false);
	//console.log("waveform loadAudioBlob completed");
    }

    async loadAudioURL(audioFile, timeChunks) {
	//console.log("waveform loadAudioURL", audioFile, timeChunks);
	await this.wavesurfer.regions.clear();
	await this.wavesurfer.load(audioFile);
	await this.loadChunks(timeChunks, false);
	//console.log("waveform loadAudioURL completed");
    }

    loadChunks(chunks, clearBefore) {
	console.log("loadChunks", chunks);
	if (clearBefore) {
	    this.wavesurfer.clearRegions();
	}
	for (let i in chunks) {
	    let chunk = chunks[i];			
	    let added = this.wavesurfer.addRegion({
		id: chunk.uuid,
		start: chunk.start / 1000.0,
		end: chunk.end / 1000.0,
		color: this.defaultRegionBackground,
		drag: false, // disable moving
	    });
	    //console.log("loadChunks debug added", added.element.title, added.id);
	}
	this.setSelectedIndex(0, false);
    }

    region2chunk(region) {
	let chunk = {
	    start: parseInt(region.start * 1000),
	    end: parseInt(region.end * 1000),
	    uuid: region.uuid,
	}
	return chunk;
    }

    getChunks() {
	//console.log("getChunks");
	let res = [];
	let regions = this.listRegions();
	for (let i = 0; i < regions.length; i++) {
	    let region = regions[i];
	    let chunk = this.region2chunk(region);
	    res.push(chunk);
	}
	return res;
    }

    getRegionFromUUID(uuid) {
	let regions = this.listRegions();
	for (let i = 0; i < regions.length; i++) {
	    let region = regions[i];
	    if (region.uuid === uuid) {
		let res = {
		    start: this.roundToInt(region.start * 1000),
		    end: this.roundToInt(region.end * 1000),
		    uuid: region.uuid,
		};
		return res;
	    }
	}
    }

    getRegion(index) {
	let regions = this.listRegions();
	if (regions.length > index) {
	    let region = regions[index];
	    let res = {
		start: this.roundToInt(region.start * 1000),
		end: this.roundToInt(region.end * 1000),
		uuid: region.uuid,
	    };
	    return res;
	}
    }

    playRegion(region) {
	console.log("playRegion", region);
	//HB this.continuousPlay = false;
	this.wavesurfer.play(region.start, region.end);
	//region.play();
    }

    updateRegion(index, start, end) {
	let regions = this.listRegions();
	if (regions.length > index) {
	    let region = regions[index];
	    region.start = start / 1000.0;
	    region.end = end / 1000.0;
	    region.update({ start: region.start, end: region.end });
	}
    }

    moveStartForRegionIndex(index, moveAmountInMilliseconds) {
	let regions = this.listRegions();
	if (regions.length > index) {
	    let region = regions[index];
	    let newStart = region.start + (moveAmountInMilliseconds / 1000.0)
	    if (newStart < 0)
		newStart = 0.0;
	    if (newStart >= region.end)
		newStart = region.end;
	    region.start = newStart;
	    region.update({ end: region.end, start: newStart });
	}
    }

    moveEndForRegionIndex(index, moveAmountInMilliseconds) {
	let regions = this.listRegions();
	if (regions.length > index) {
	    let region = regions[index];
	    let newEnd = region.end + (moveAmountInMilliseconds / 1000.0)
	    if (newEnd < this.length)
		newEnd = this.length - 0.5; // TODO
	    if (newEnd <= region.start)
		newEnd = region.start;
	    region.end = newEnd;
	    region.update({ start: region.start, end: newEnd });
	}
    }

    getSelectedRegion() {
	this.debug("getSelectedRegion");
	let regions = this.listRegions();
	for (let id in regions) {
	    let region = regions[id];
	    if (region.element.classList.contains("selected")) {
		//console.log("getSelectedRegion", region);
		return region;
	    }
	}
    }

    getSelectedRegionIndex() {
	this.debug("getSelectedRegionIndex");
	let regions = this.listRegions();
	let i = 0;
	for (let id in regions) {
	    let region = regions[id];
	    if (region.element.classList.contains("selected")) {
		//console.log("getSelectedRegionIndex", region);
		return i;
	    }
	    i++;
	}
	return -1;
    }

    unselect(regionElements) {
	for (let j in regionElements) {
	    let e = regionElements[j];
	    let id = e.getAttribute("data-id")
	    if (e.localName === "region") {
		e.classList.remove("selected");
		//HB 0728
		//e.style["background-color"] = this.defaultRegionBackground;
		//e.style["background-color"] = 'blue';
	    }
	}
    }

    setSelectedRegion(region, playMode, blockOnSelectedRegionChange) {
	if ( blockOnSelectedRegionChange==null) {
	    blockOnSelectedRegionChange = false;
	}
	console.log("setSelectedRegion", region.uuid, playMode, blockOnSelectedRegionChange);
	this.debug("setSelectedRegion", region.uuid, region, playMode);
	console.log("selectedRegionIndex: " + this.getSelectedRegionIndex())


	//HB 230125 Maybe move this to after unselect ??
	region.element.classList.add("selected");
	region.element.style["background-color"] = this.selectedRegionBackground;

	console.log("BEFORE unselect siblings: " + LIB.siblings(region.element, false))
	console.log("Region classlist: " + region.element.classList);
	console.log("selectedRegionIndex: " + this.getSelectedRegionIndex())

	//HB 230115 Fix for wavesurfer v6
	//With wavesurfer.js v6 there is a problem here
	//something to do with Proxy, domElement https://github.com/katspaugh/wavesurfer.js/blob/master/UPGRADE.md
	//this.unselect(LIB.siblings(region.element, false));
	this.unselect(LIB.siblings(region.element.domElement, false));

	//HB move to here ??
	//region.element.classList.add("selected");
	//region.element.style["background-color"] = this.selectedRegionBackground;
	
	console.log("AFTER unselect siblings")
	console.log("Region classlist: " + region.element.classList);
	console.log("selectedRegionIndex: " + this.getSelectedRegionIndex())
	

	if (playMode === true) {
	    console.log("calling playRegion");
	    this.playRegion(region);
	}
	else if (this.autoplayEnabledFunc() && playMode !== false) {
	    console.log("calling playRegion");
	    this.playRegion(region);
	}
	if (this.onSelectedRegionChange && !blockOnSelectedRegionChange) {
	    console.log("calling app.onSelectedRegionChange");
	    console.log("selectedRegionIndex: " + this.getSelectedRegionIndex())
	    this.onSelectedRegionChange(region.uuid);
	}
	console.log("selectedRegionIndex: " + this.getSelectedRegionIndex())
	console.log("END setSelectedRegion");
    }

    setSelectedIndex(index, playMode) {
	this.debug("setSelectedIndex", index);
	let regions = this.listRegions();
	if (regions.length > index)
	    this.setSelectedRegion(regions[index], playMode);
    }

    // list regions sorted by start time
    listRegions() {
	let regions = Object.values(this.wavesurfer.regions.list);
	regions.sort(function (a, b) { return a.start - b.start });
	for (let i = 0; i < regions.length; i++) {
	    let region = regions[i];
	    region.uuid = region.id;
	}
	return regions;
    }

    selectPrevRegion() {
	let regions = this.listRegions();
	let lastSelected;
	for (let i = regions.length - 1; i >= 0; i--) {
	    let region = regions[i];
	    this.debug("selectPrevRegion", regions.length, i, region, lastSelected);
	    if (lastSelected) {
		this.setSelectedRegion(region);
		return true;
	    }
	    if (region.element.classList.contains("selected"))
		lastSelected = region;
	}
	return false;
    }

    selectNextRegion() {
	let regions = this.listRegions();
	let lastSelected;
	for (let i = 0; i < regions.length; i++) {
	    let region = regions[i];
	    this.debug("selectNextRegion", region, lastSelected);
	    if (lastSelected) {
		this.setSelectedRegion(region);
		return true;
	    }
	    if (region.element.classList.contains("selected"))
		lastSelected = region;
	}
	return false;
    }

    clear() {
	let wsPane = document.getElementById("waveform-pane");
	let h = wsPane.offsetHeight;
	let w = wsPane.offsetWidth;
	this.wavesurfer.regions.clear();
	// this.wavesurfer.timeline.destroy();
	// this.wavesurfer.spectrogram.destroy();
	this.wavesurfer.empty();
	//this.wavesurfer.cursorColor = "transparent";
	// wsPane.style["height"] = h + "px";
	// wsPane.style["width"] = w + "px";
    }

    // LIB

    debug(msg) {
	if (this.options.debug) console.log("waveform debug", msg);
    }

    logEvent(evt) {
	this.debug("LOG EVENT | type: " + evt.type + ", element id:" + evt.target.id, evt);
    }

    floatWithDecimals(f0, decimalCount) {
	let f = Number((f0).toFixed(decimalCount));
	let res = f + "";
	if (!res.includes("."))
	    return res + ".00";
	if (res.match(/[.][0-9]$/g))
	    return res + "0";
	// if (res.match(/[.][0-9][0-9]$/g))
	//     return res + "0";
	else return res;
    }

    roundToInt(f) {
	return Number((f).toFixed(0));
    }

}

