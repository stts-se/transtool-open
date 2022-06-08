'use strict';

const baseURL = window.location.protocol + '//' + window.location.host + window.location.pathname.replace(/\/$/g, "");
const wsBase = baseURL.replace(/^http/, "ws");
const clientID = LIB.uuidv4();
let ws;

// https://www.w3schools.com/htmL/html_colors_hsl.asp#:~:text=%20HTML%20HSL%20and%20HSLA%20Colors%20%201,Alpha%20channel%20-%20which%20specifies%20the...%20More
// const transparentBlue = 'hsla(200, 50%, 70%, 0.4)';
// const transparentGreen = 'hsla(120, 100%, 75%, 0.3)';
// const transparentGrey = 'hsla(0, 0%, 75%, 0.3)';
// const transparentOrange = 'hsla(39, 100%, 50%, 0.3)';
const transparentBlue = 'hsla(200, 50%, 70%, 0.6)';
const transparentGreen = 'hsla(120, 100%, 75%, 0.5)';
const transparentGrey = 'hsla(0, 0%, 75%, 0.5)';
const transparentOrange = 'hsla(39, 100%, 50%, 0.5)';

let gloptions = {
    boundaryMovementShort: 50,
    boundaryMovementLong: 100,
    defaultSetStatus: 'ok',
    defaultRequestStatus: 'unchecked',
    defaultRequestSource: 'any',
    defaultRequestAudioFile: 'any',
    //context: -1,
}

let enabled = false;
let waveform;

// type: payload.AnnotationWithAudioData
let pageCache;
let chunkCache;

let debugVar;

function logWarning(msg) {
    let div = logMessage("warning", msg);
    div.style.color = "orange";
}

function logError(msg) {
    let div = logMessage("error", msg);
    div.style.color = "red";
}

function logMessage(arg1, arg2) {
    let level = "info";
    let msg = arg1;
    if (arg2) {
        level = arg1;
        msg = arg2;
    }
    msg = "[" + level + "] " + msg;
    let div = document.createElement("div");
    div.innerText = new Date().toLocaleTimeString("sv-SE") + " " + msg;
    messages.prepend(div);
    return div;
}

function lockGUI() {
    setEnabled(false);
    enableStart(false);
}

function enableStart(enable) {
    if (enable) {
        document.getElementById("start").disabled = false;
        document.getElementById("start").classList.remove("disabled");
    } else {
        document.getElementById("start").disabled = true;
        document.getElementById("start").classList.add("disabled");
    }
}

function setEnabled(enable) {
    document.getElementById("unlock-all").disabled = false;
    document.getElementById("unlock-all").classList.remove("disabled");

    if (waveform)
        waveform.setEnabled(enable);

    enabled = enable;
    let buttons = [
        document.getElementById("save-skip"),
        document.getElementById("save-ok"),
        document.getElementById("save-progress"),
        document.getElementById("save-skip-next"),
        document.getElementById("save-ok-next"),
        document.getElementById("play-all"),
        document.getElementById("play-selected"),
        document.getElementById("play-right"),
        document.getElementById("play-left"),
        //document.getElementById("reset"),
        document.getElementById("quit"),
        document.getElementById("next_page"),
        document.getElementById("prev_page"),
        document.getElementById("first_page"),
        document.getElementById("last_page"),
        document.getElementById("next_page_any"),
        document.getElementById("prev_page_any"),
        document.getElementById("asr-request"),
        document.getElementById("delete-selected"),
        document.getElementById("add_abbrev"),
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
        // document.getElementById("start").disabled = true;
        // document.getElementById("start").classList.add("disabled");
        document.getElementById("comment").removeAttribute("readonly");
        document.getElementById("editor-text-area").removeAttribute("readonly");
    } else {
        document.getElementById("comment").setAttribute("readonly", "readonly");
        document.getElementById("editor-text-area").setAttribute("readonly", "readonly");
        for (let i = 0; i < buttons.length; i++) {
            let btn = buttons[i];
            if (btn) {
                btn.classList.add("disabled");
                btn.disabled = true;
            }
        }
        // document.getElementById("start").disabled = false;
        // document.getElementById("start").classList.remove("disabled");
    }
    enableStart(!enable);
}

function getFromURLParamsOrLocalStorage(paramName, urlParams) {
    if (!urlParams)
	urlParams = new URLSearchParams(window.location.search);
    //console.log("param", paramName, urlParams.get(paramName), localStorage.getItem(paramName));
    if (urlParams.get(paramName)) {
	return urlParams.get(paramName);
    }
    else if (localStorage.getItem(paramName)) {
	return localStorage.getItem(paramName);
    }
    else
	return null;
}

async function loadAudioBlob(blob, chunks) {
    let wfRegions = [];
    for (let i = 0; i < chunks.length; i++) {
        let ch = chunks[i];
        wfRegions.push({
            start: ch.start - pageCache.offset,
            end: ch.end - pageCache.offset,
            uuid: ch.uuid,
        });
    }
    waveform.loadAudioBlob(blob, wfRegions);
}

function listAvailableAudioFiles() {
    let request = {
        'client_id': clientID,
        'message_type': 'list-db-audio-files-request'
    };
    ws.send(JSON.stringify(request));
    logMessage("Sent request to list available audio files");
}

function sendToASR(wfRegion) {
    console.log("sendToASR debug wfRegion", wfRegion);
    let region = wfRegion;
    region.start = region.start + pageCache.offset;
    region.end = region.end + pageCache.offset;
    let pageID = pageCache.page.id;

    const len = region.end - region.start;
    if (len >= 60000) {
        //logError("Cannot run ASR on pages over 1 min (selected page is " + len + "ms)")
        console.log("Cannot run ASR on pages over 1 min (selected page is " + len + "ms)");
        return;
    }


    let payload = {
        "page_id": pageID,
        "uuid": region.uuid,
        "chunk": {
            "start": region.start,
            "end": region.end
        },
        "lang": "sv-SE", // todo hardwired
    };
    //console.log("payload", JSON.stringify(payload));


    let request = {
        'client_id': clientID,
        'message_type': 'asr-request',
        'payload': JSON.stringify(payload),
    };
    ws.send(JSON.stringify(request));
    logMessage("Sent ASR request for page " + pageID + " (" + region.start + "-" + region.end + " ms)");
}

document.getElementById("clear_local_storage").addEventListener("click", function (evt) {
    localStorage.clear();
});

document.getElementById("add_abbrev").addEventListener("click", function (evt) {
    abbrevPopUp(evt);
});

document.getElementById("delete-selected").addEventListener("click", function (evt) {
    deleteSelectedChunk();
});

document.getElementById("asr-request").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        //console.log(evt.target.id, "clicked");
        let ri = waveform.getSelectedRegionIndex();
        if (ri >= 0) {
            let region = waveform.getRegion(ri);
            if (region) {
                const oldText = document.getElementById("editor-text-area").innerText.trim();
                if (oldText.length > 0) {
                    let overwrite = confirm("Overwrite existing transcription with new ASR?");
                    if (!overwrite) {
                        return;
                    }
                }

                sendToASR(region);
                //document.getElementById("play-selected").click();
            }
        } else {
            logMessage("No selected chunk to send to ASR");
        }
    }
});

document.getElementById("play-selected").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        //console.log(evt.target.id, "clicked");
        let ri = waveform.getSelectedRegionIndex();
        if (ri >= 0) {
            waveform.playRegionIndex(ri);
        }
        else
            logMessage("No selected chunk to play");
    }
});

document.getElementById("waveform-playpause").addEventListener("click", function (evt) {
    console.log("waveform-playpause");
    waveform.logEvent(evt);
    waveform.continuousPlay = true;
    if (waveform.wavesurfer.isPlaying())
        waveform.wavesurfer.pause();
    else
        waveform.wavesurfer.play();
});


const onUserAddedRegion = function(wfRegion) {
    console.log("app.js onUserAddedRegion", wfRegion);
    if (!chunkCache[wfRegion.uuid]) {
	if (waveform.wavesurfer.isPlaying())
            waveform.wavesurfer.pause();
	document.getElementById("editor-text-area").innerText = "";

	let ch = waveform.region2chunk(wfRegion);
	let timestamp = new Date().toLocaleString("sv-SE");
	let status = {
            name: "unchecked",
            source: document.getElementById("username").innerText,
            timestamp: timestamp,
	}
	ch.current_status = status;
	ch.status_history = [];
	chunkCache[ch.uuid] = ch;
	updateStatusColors();
	updateStatusDisplay("chunk", ch.uuid, status);
	//waveform.setSelectedRegion(wfRegion);
    }
}

const onSelectedRegionChange = function (uuid) {
    document.getElementById("asr_info").innerText = "";
    document.getElementById("reverse_expansion").innerText = "";

    let wfRegion = waveform.getRegionFromUUID(uuid);
    if (!wfRegion)
        return;
    document.getElementById("editor-text-area").removeAttribute("readonly");
    //console.log("onSelectedRegionChange", uuid, wfRegion);
    if (wfRegion) {
        // TODO 
        //sendToASR(wfRegion);
    }
    if (document.getElementById("autoplay").checked) {
	//waveform.playRegionWithUUID(uuid);
    } else if (waveform.wavesurfer.isPlaying()) {
        waveform.wavesurfer.pause();
    }
    document.getElementById("editor-text-area").innerText = "";

    if (chunkCache[uuid]) {
        let ch = chunkCache[uuid];
        if (ch.trans) {
            document.getElementById("editor-text-area").innerText = ch.trans;
            document.getElementById("editor-text-area").focus();
        }
        if (!ch.current_status) {
            throw new Error("No current_status for chunk " + JSON.stringify(ch));
        }
        updateStatusDisplay("chunk", ch.uuid, ch.current_status);
    } else {
        let ch = waveform.region2chunk(wfRegion);
        let timestamp = new Date().toLocaleString("sv-SE");
        let status = {
            name: "unchecked",
            source: document.getElementById("username").innerText,
            timestamp: timestamp,
        }
        ch.current_status = status;
        ch.status_history = [];
        chunkCache[ch.uuid] = ch;
        updateStatusDisplay("chunk", ch.uuid, status);
    }

    if (document.getElementById("autoasr").checked && document.getElementById("editor-text-area").innerText.trim().length == 0)
        document.getElementById("asr-request").click();
    // else if (document.getElementById("autoplay").checked && !document.getElementById("autoasr").checked)
    //     document.getElementById("play-selected").click();
    updateStatusColors();
}

function debugWithStackTrace(msg) {
    try {
    	throw new Error();
    } catch (e) {
	if (msg)
    	    console.log(msg, e);
	else
    	    console.log(e);	    
    }
}

function updateStatusColors() {
    console.log("updateStatusColors");
    let regions = waveform.listRegions();
    for (let i = 0; i < regions.length; i++) {
        let region = regions[i];
        let statusText;
        if (chunkCache[region.uuid]) {
            let chunk = chunkCache[region.uuid];
            if (!chunk.current_status) {
                throw new Error("No current_status for chunk " + JSON.stringify(chunk));
            }
            if (region.selected) {
                region.color = transparentBlue;
                //region.element.classList.add("selected");
                statusText = chunk.current_status.name + " (selected)";
            } else if (chunk.current_status) {
                if (chunk.current_status.name === "unchecked") {
                    statusText = "unch";
                    region.color = transparentGrey;
                    //region.element.classList.add("unchecked");
                } else {
                    statusText = chunk.current_status.name;
                    if (chunk.current_status.name.startsWith("ok")) {
                        region.color = transparentGreen;
                        //region.element.classList.add("ok");
                    } else if (chunk.current_status.name === "skip") {
                        region.color = transparentOrange;
                        //region.element.classList.add("skip");
                    } else {
                        region.color = transparentGrey;
                        //region.element.classList.add("unchecked");
                    }
                }
            } else {
                region.color = transparentGrey;
                //region.element.classList.add("unchecked");
            }
        } else {
            region.color = transparentGrey;
            //region.element.classList.add("unchecked");
            statusText = "new";
        }
        region.element.style["background-color"] = region.color;

        let children = region.element.childNodes;
        let text;
        for (let i = 0; i < children.length; i++) {
            let child = children[i];
            if (child.localName === "span") {
                text = child;
            }
        }
        if (!text)
            text = document.createElement("span");
        region.element.style["text-align"] = "center";
        text.style["text-align"] = "center";
        text.innerHTML = statusText;
        region.element.appendChild(text);
    }
}

document.getElementById("waveform-skipforward").addEventListener("click", function (evt) {
    //console.log("waveform-skipforward", evt);
    waveform.logEvent(evt);
    if (!evt.target.disabled && waveform.getSelectedRegionIndex() >= 0) {
        savePage({ status: "in progress" });
        waveform.selectNextRegion();
    }
});


document.getElementById("waveform-skipback").addEventListener("click", function (evt) {
    waveform.logEvent(evt);
    if (!evt.target.disabled && waveform.getSelectedRegionIndex() >= 0) {
        savePage({ status: "in progress" });
        waveform.selectPrevRegion();
    }
});

if (document.getElementById("waveform-skiptolast")) {
    document.getElementById("waveform-skiptolast").addEventListener("click", function (evt) {
        waveform.logEvent(evt);
        if (!evt.target.disabled) {
            let regions = waveform.listRegions();
            if (regions.length > 0) {
                savePage({ status: "in progress" });
                waveform.setSelectedRegion(regions[regions.length - 1]);
            }
        }
    });
}


if (document.getElementById("waveform-skiptofirst")) {
    document.getElementById("waveform-skiptofirst").addEventListener("click", function (evt) {
        waveform.logEvent(evt);
        if (!evt.target.disabled) {
            if (waveform.getSelectedRegionIndex() != 0) {
                savePage({ status: "in progress" });
                waveform.setSelectedIndex(0);
            }
        }
    });
}

document.getElementById("save-skip-next").addEventListener("click", function (evt) {
    if (!evt.target.disabled && waveform.getSelectedRegionIndex() >= 0) {
        saveCurrentChunk({ status: "skip", moveRight: true });
        // saveUnlockAndNext({ status: "skip", stepSize: 1 });
    }
});
document.getElementById("save-ok-next").addEventListener("click", function (evt) {
    if (!evt.target.disabled && waveform.getSelectedRegionIndex() >= 0) {
        saveCurrentChunk({ status: document.getElementById("setstatus").value, moveRight: true });
        // saveUnlockAndNext({ status: "ok", stepSize: 1 });
    }
});
document.getElementById("save-ok").addEventListener("click", function (evt) {
    if (!evt.target.disabled && waveform.getSelectedRegionIndex() >= 0)
        saveCurrentChunk({ status: document.getElementById("setstatus").value });
});
document.getElementById("save-progress").addEventListener("click", function (evt) {
    //console.log("save-progress clicked");
    if (!evt.target.disabled && waveform.getSelectedRegionIndex() >= 0)
        savePage({ status: "in progress" });
});

if (document.getElementById("first_page")) {
    document.getElementById("first_page").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ requestIndex: "first" });
    });
}
if (document.getElementById("last_page")) {
    document.getElementById("last_page").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ requestIndex: "last" });
    });
}
if (document.getElementById("start")) {
    document.getElementById("start").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ stepSize: 1 });
    });
}
if (document.getElementById("next_page")) {
    document.getElementById("next_page").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ stepSize: 1 });
    });
}
if (document.getElementById("prev_page")) {
    document.getElementById("prev_page").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ stepSize: -1 });
    });
}

if (document.getElementById("prev_page_any")) {
    document.getElementById("prev_page_any").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ stepSize: -1, requestStatus: "any", requestSource: "any", requestAudioFile: "any", requestTransRE: "" });
    });
}
if (document.getElementById("next_page_any")) {
    document.getElementById("next_page_any").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ stepSize: 1, requestStatus: "any", requestSource: "any", requestAudioFile: "any", requestTransRE: "" });
    });
}

function clear() {
    if (waveform)
        waveform.clear();
    document.getElementById("comment").value = "";
    document.getElementById("editor-text-area").innerText = "";
    //document.getElementById("labels").innerText = "";
    if (document.getElementById("current_page_status"))
        document.getElementById("current_page_status").innerText = "";
    document.getElementById("current_chunk_status").innerText = "";
    if (document.getElementById("current_page_status_div"))
        document.getElementById("current_page_status_div").style.borderColor = "";
    document.getElementById("current_chunk_status_div").style.borderColor = "";
    document.getElementById("page_info").innerHTML = "&nbsp;";
}

document.getElementById("reset").addEventListener("click", function (evt) {
    alert("reset is not fully functional right now");
    return;
    if (!evt.target.disabled) {
        waveform.updateRegion(0, pageCache.chunk.start, pageCache.chunk.end); // TODO - does not work
        if (pageCache.comment)
            document.getElementById("comment").value = pageCache.comment;
        else
            document.getElementById("comment").value = "";
        // TODO reset trans area
    }
});
document.getElementById("quit").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        unlockCurrentPage();
        setEnabled(false);
        clear();
    }
});

document.getElementById("unlock-all").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        unlockAll();
        setEnabled(false);
        clear();
    }
});

document.getElementById("autoplayonasr").addEventListener("change", function (evt) {
    //console.log("change", evt.target);
    localStorage.setItem("autoplayonasr", evt.target.checked);
});
document.getElementById("autoasr").addEventListener("change", function (evt) {
    //console.log("change", evt.target);
    localStorage.setItem("autoasr", evt.target.checked);
});
document.getElementById("autoplay").addEventListener("change", function (evt) {
    //console.log("change", evt.target);
    localStorage.setItem("autoplay", evt.target.checked);
});

document.getElementById("setstatus").addEventListener("change", function (evt) {
    //console.log("change", evt.target);
    let setstatus = evt.target.value;
    localStorage.setItem("set_status", setstatus);
    if (evt.target.value !== gloptions.defaultSetStatus) {
	evt.target.classList.add("search_active");
    } else {
	evt.target.classList.remove("search_active");
    }
    
    document.getElementById("save-ok").innerText = setstatus;
    document.getElementById("save-ok-next").innerText = setstatus + "+next";
});

document.getElementById("requeststatus").addEventListener("change", function (evt) {
    //console.log("change", evt.target);
    localStorage.setItem("request_status", evt.target.value);
    if (evt.target.value !== gloptions.defaultRequestStatus) {
	evt.target.classList.add("search_active");
    } else {
	evt.target.classList.remove("search_active");
    }
});
document.getElementById("requestsource").addEventListener("change", function (evt) {
    //console.log("change", evt.target);
    localStorage.setItem("request_source", evt.target.value);
    if (evt.target.value !== gloptions.defaultRequestSource) {
	evt.target.classList.add("search_active");
    } else {
	evt.target.classList.remove("search_active");
    }
});
document.getElementById("requestaudiofile").addEventListener("change", function (evt) {
    //console.log("change", evt.target);
    localStorage.setItem("request_audio_file", evt.target.value);
    if (evt.target.value !== gloptions.defaultRequestAudioFile) {
	evt.target.classList.add("search_active");
    } else {
	evt.target.classList.remove("search_active");
    }
    document.getElementById("load_stats").click();
    evt.target.title = "Audio: " + evt.target.value;
});

document.getElementById("load_stats").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        let request = {
            'client_id': clientID,
            'message_type': 'stats',
        };
        ws.send(JSON.stringify(request));
    }
});

document.getElementById("move-left2left-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(waveform.getSelectedRegionIndex(), -gloptions.boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-left2right-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(waveform.getSelectedRegionIndex(), gloptions.boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2left-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(waveform.getSelectedRegionIndex(), -gloptions.boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2right-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(waveform.getSelectedRegionIndex(), gloptions.boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});

document.getElementById("move-left2left-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(waveform.getSelectedRegionIndex(), -gloptions.boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-left2right-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(waveform.getSelectedRegionIndex(), gloptions.boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2left-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(waveform.getSelectedRegionIndex(), -gloptions.boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2right-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(waveform.getSelectedRegionIndex(), gloptions.boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});

//HB current_chunk_info (det är 'chunk' i vågformen)
//HB här kanske man kan lägga till "viewWindow" också? Eller så är totChunks bara totalt antal i "page"
// statusLevel is chunk or page
function updateStatusDisplay(statusLevel, uuid, status) {
    // console.log("updateStatusDisplay", statusLevel, status)
    const regions = waveform.listRegions();
    const totChunks = regions.length;
    let currIndex = -1;
    for (let i=0;i<regions.length;i++) {
	const r = regions[i];
	if (r.id === uuid)
	    currIndex = i+1;
    }
    let statusText = "#" + currIndex + "/" + totChunks + " | " + status.name;
    let statusDiv = document.getElementById("current_" + statusLevel + "_status_div");
    if (status.name.startsWith("ok"))
        statusDiv.style.borderColor = "lightgreen";
    else if (status.name === "bad sample")
        statusDiv.style.borderColor = "#ff5757";
    else if (status.name === "skip")
        statusDiv.style.borderColor = "orange";
    else if (status.name === "in progress")
        statusDiv.style.borderColor = "lightblue";
    else if (status.name === "unchecked")
        statusDiv.style.borderColor = "lightgrey";
    else
        statusDiv.style.borderColor = "black";

    if (status.source)
        statusText += " (" + status.source + ")";
    if (status.timestamp)
        statusText += " | " + status.timestamp;
    document.getElementById("current_" + statusLevel + "_status").innerText = statusText;
}

function displayAnnotationWithAudioData(anno) {
    clear();
    lockGUI();
    // https://stackoverflow.com/questions/16245767/creating-a-blob-from-a-base64-string-in-javascript#16245768
    let byteCharacters = atob(anno.base64audio);
    let byteNumbers = new Array(byteCharacters.length);
    for (let i = 0; i < byteCharacters.length; i++) {
        byteNumbers[i] = byteCharacters.charCodeAt(i);
    }
    let byteArray = new Uint8Array(byteNumbers);

    pageCache = anno;
    pageCache.base64audio = null; // no need to cache the audio blob
    chunkCache = {};
    for (let i = 0; i < anno.chunks.length; i++) {
        let chunk = anno.chunks[i];
        chunkCache[chunk.uuid] = chunk;
    }
    console.log("res => cache", pageCache, chunkCache);

    let blob = new Blob([byteArray], { 'type': anno.file_type });
    loadAudioBlob(blob, anno.chunks);
    let prettyLen = time_convert(anno.page.end - anno.page.start);
    //let prettyLen = (anno.page.end - anno.page.start) + " ms";
    //HB page_info - det kan bli en lista av sökta pages så småningom
    document.getElementById("page_info").innerHTML = anno.index + " | " + anno.page.id + " | " + prettyLen + " | <span title='Location in full audio file'>" + anno.page.start + " - " + anno.page.end + "</span>";

    // status info + color code
    //updateStatusDisplay("page", anno.current_status);

    // comment
    if (anno.comment)
        document.getElementById("comment").value = anno.comment;

    // labels => integrated as status
    // if (anno.labels && anno.labels.length > 0)
    //     document.getElementById("labels").innerText = anno.labels;
    // else
    //     document.getElementById("labels").innerText = "none";

    // select first chunk matching request criteria -- does not work since it will be called _before_ audio+chunks are loaded
    // selectNextChunkMatchingRequestCriteria()

    setEnabled(true);
    //document.getElementById("editor-text-area").focus();

    logMessage("Loaded annotation with audio for page " + anno.page.id + " from server");
}

function displayStats(stats) {
    logMessage("Received stats from server");
    console.log("Received stats from server", stats);
    let timestamp = new Date().toLocaleTimeString("sv-SE");
    document.getElementById("stats_timestamp").innerText = timestamp;

    let selectedAudio = document.getElementById("requestaudiofile").value;
    if (!selectedAudio || selectedAudio === "any") {
	selectedAudio = "all";
	document.getElementById("stats_for_audio").innerText = "Showing stats for all audio";
    } else {
	document.getElementById("stats_for_audio").innerHTML = "Showing stats for audio <em>" + selectedAudio + "</em>";
    }

    if (!stats["page"][selectedAudio]) {
	logError("No page stats for audio " + selectedAudio);
	return;
    }
    if (!stats["chunk"][selectedAudio]) {
	logError("No chunk stats for audio " + selectedAudio);
	return;
    }
    
    let ele = document.getElementById("stats");
    ele.innerHTML = '<thead><tr><td colspan="2"><em>Pages</em></td><td width="35px"></td><td colspan="2"><em>Chunks</em></td></tr></thead>';

    let pKeys = Object.keys(stats["page"][selectedAudio]);
    pKeys.sort();
    let cKeys = Object.keys(stats["chunk"][selectedAudio]);
    cKeys.sort();

    for (let i = 0; i < pKeys.length || i < cKeys.length; i++) {
        let pKey = "", pVal = "";
        if (i < pKeys.length) {
            pKey = pKeys[i];
            pVal = stats["page"][selectedAudio][pKey];
        }
        let cKey = "", cVal = "";
        if (i < cKeys.length) {
            cKey = cKeys[i];
            cVal = stats["chunk"][selectedAudio][cKey];
        }
        let tr = document.createElement("tr");
        let td1 = document.createElement("td");
        let td2 = document.createElement("td");
        let td3 = document.createElement("td");
        let td4 = document.createElement("td");
        let td5 = document.createElement("td");
        td2.style["text-align"] = "right";
        td5.style["text-align"] = "right";
        td1.innerHTML = pKey;
        td2.innerHTML = pVal;
        td4.innerHTML = cKey;
        td5.innerHTML = cVal;
        tr.appendChild(td1);
        tr.appendChild(td2);
        tr.appendChild(td3);
        tr.appendChild(td4);
        tr.appendChild(td5);
        ele.appendChild(tr);
    }
}

function unlockCurrentPage() {
    console.log("unlockCurrentPage called")
    if (pageCache === undefined || pageCache === null || chunkCache === undefined || chunkCache === null)
        return;

    let request = {
        'client_id': clientID,
        'message_type': 'unlock',
        'payload': JSON.stringify({
            'page_id': pageCache.page.id,
            'user_name': document.getElementById("username").innerText,
        }),
    };
    ws.send(JSON.stringify(request));
}

function unlockAll() {
    console.log("unlockAll called")
    let request = {
        'client_id': clientID,
        'message_type': 'unlock_all',
        'payload': JSON.stringify({
            'user_name': document.getElementById("username").innerText,
        }),
    };
    ws.send(JSON.stringify(request));
}

document.getElementById("clear_messages").addEventListener("click", function (evt) {
    document.getElementById("messages").innerHTML = "";
});


function createQuery(stepSize, requestIndex, requestStatus, requestSource, requestAudioFile, requestTransRE) {
    let query = {
        user_name: document.getElementById("username").innerText,
    }
    if (stepSize)
        query.step_size = stepSize;
    if (requestIndex)
        query.request_index = requestIndex;
    // if (gloptions.context && gloptions.context >= 0)
    //     query.context = parseInt(gloptions.context);
    if (pageCache && pageCache !== null)
        query.curr_id = pageCache.page.id;

    query.request = {};
    if (requestStatus)
        query.request.status = requestStatus;
    else {
        query.request.status = document.getElementById("requeststatus").value;
    }
    if (requestSource)
        query.request.source = requestSource;
    else {
        query.request.source = document.getElementById("requestsource").value;
    }
    if (requestAudioFile)
        query.request.audio_file = requestAudioFile;
    else {
        query.request.audio_file = document.getElementById("requestaudiofile").value;
    }
    if (requestTransRE)
        query.request.trans_re = requestTransRE;
    else {
        query.request.trans_re = document.getElementById("requesttransre").value;
    }
    console.log("createQuery output", query);
    return query;
}

// function next(stepSize) {
//     console.log("next called")
//     let request = {
//     	'client_id': clientID,
//     	'message_type': 'next',
//     	'payload': JSON.stringify(createQuery(stepSize)),
//     };
//     ws.send(JSON.stringify(request));
// }

function computeCurrentAnnotation(options, user) {
    // let status = {
    //     source: user,
    //     name: options.status,
    //     timestamp: new Date().toLocaleString("sv-SE"),
    // }
    // let statusHistory = pageCache.status_history;
    // if (!statusHistory)
    //     statusHistory = [];
    // if (pageCache.current_status.name !== "unchecked")
    //     statusHistory.push(pageCache.current_status);

    let labels = [];
    if (options.label) {
        labels.push(options.label);
    }

    // let tmpChunkCache = {};
    // for (let j = 0; j < pageCache.chunks.length; j++) {
    //     let ch = pageCache.chunks[j];
    //     tmpChunkCache[ch.uuid] = ch;
    // }
    //console.log("computeCurrentAnnotation debug tmpChunkCache", tmpChunkCache);
    let wfChunks = waveform.getChunks();
    let chunks = [];
    for (let i = 0; i < wfChunks.length; i++) {
        let chunk = {
            start: wfChunks[i].start + pageCache.offset,
            end: wfChunks[i].end + pageCache.offset,
            uuid: wfChunks[i].uuid,
        }
        if (chunkCache[chunk.uuid]) {
            let cachedChunk = chunkCache[chunk.uuid];
            chunk.trans = cachedChunk.trans;
            chunk.current_status = cachedChunk.current_status;
            chunk.status_history = cachedChunk.status_history;
        } else {
            throw new Error("No status cache for chunk " + JSON.stringify(chunk));
            let status = {
                source: user,
                name: "unchecked",
                timestamp: new Date().toLocaleString("sv-SE"),
            }
            chunk.current_status = status;
            chunk.status_history = [];
            chunkCache[chunk.uuid] = chunk;
        }
        chunks.push(chunk);
    }

    //console.log("computeCurrentAnnotation debug chunks", chunks);
    let annotation = {
        page: pageCache.page,
        chunks: chunks,
        // current_status: status,
        // status_history: statusHistory,
        labels: labels,
        comment: document.getElementById("comment").value,
        index: pageCache.index,
    };
    // if (options.status === "derive") {
    //     annotation.current_status.name = derivePageStatus(annotation);
    // }
    //console.log("debug annotation", annotation);
    return annotation;
}

function selectNextChunkMatchingRequestCriteria() {
    let query = createQuery(1, .1,
        document.getElementById("requeststatus").value,
        document.getElementById("requestsource").value,
        document.getElementById("requestaudiofile").value,
        document.getElementById("requesttransre").value);
    console.log("selectNextChunkMatching", query);
    // if (query.request.status === "any" && query.request.source === "any" && query.request.transre === "")
    //     return waveform.selectNextRegion();
    let wfcs = waveform.getChunks();
    let selectedIndex = waveform.getSelectedRegionIndex();
    if (selectedIndex >= wfcs.length) { // at last chunk
        return false;
    }
    for (let i = selectedIndex + 1; i < wfcs.length; i++) {
        let wfc = wfcs[i];
        let chunk = chunkCache[wfc.uuid];
        if (chunk) {
            let status = chunk.current_status;
           
            let statusMatch = false;
            if (query.request.status === "any")
                statusMatch = true;
            else if (!status)
                statusMatch = (query.request.status === "unchecked");
            else if (status.name === query.request.status)
                statusMatch = true;
            else if (query.request.status === "unchecked")
                statusMatch = (status.name === "" || status.name === "unchecked")
            else if (query.request.status === "checked")
                statusMatch = (status.name !== "" && status.name !== "unchecked");
           
            let sourceMatch = false;
            if (query.request.source === "any")
                sourceMatch = true;
            else if (!status)
                sourceMatch = (query.request.source === "any");
            else 
                sourceMatch = (status.source === query.request.source);
            
            let transREMatch = false;
            if (chunk.trans && query.request.trans_re.trim().length)
                transREMatch = chunk.trans.match(query.request.trans_re.trim());
            else
                transREMatch = true;
            console.log("transREMatch", transREMatch, query.request.transre);
            if (statusMatch && sourceMatch && transREMatch) {
                waveform.setSelectedIndex(i);
                return true;
            }
        }
    }
    return false;
}

function saveCurrentChunk(options) {
    let user = document.getElementById("username").innerText;
    let status = {
        source: user,
        name: options.status,
        timestamp: new Date().toLocaleString("sv-SE"),
    }
    console.log("debug saveCurrentChunk pageCache", pageCache, chunkCache);
    let ch = cacheActiveTranscription(status);
    updateStatusDisplay("chunk", ch.uuid, status);
    let defaultSaveStatus = "in progress";

    if (options.moveRight) {
        //let requestStatus = document.getElementById("requeststatus").value;
        if (selectNextChunkMatchingRequestCriteria()) {
            savePage({ status: defaultSaveStatus });
        } else {
	    let currentChunks = computeCurrentAnnotation(options, user).chunks;
	    let nUnchecked = 0;
	    for (let i=0;i<currentChunks.length;i++) {
		let chunk = currentChunks[i];
		if (chunk.current_status && chunk.current_status.name === "unchecked")
		    nUnchecked++;
	    }
	    let nUncheckedChunksText = "Page contains " + nUnchecked + " unchecked chunks.\n";
	    if (nUnchecked === 0)
		nUncheckedChunksText = "";
	    if (nUnchecked === 1)
		nUncheckedChunksText = "Page contains one unchecked chunk.\n";
	    if (nUnchecked === 2)
		nUncheckedChunksText = "Page contains two unchecked chunks.\n";
            let switchPages = confirm("There are no more chunks matching request query on this page.\n" + nUncheckedChunksText + "Save current page, and move to next?");
            if (switchPages) {
                saveUnlockAndNext({ status: "derive", stepSize: 1 });
            } else {
                savePage({ status: defaultSaveStatus });
            }
        }
    } else {
        savePage({ status: defaultSaveStatus });
    }
    updateStatusColors();
}

function savePage(options) {
    console.log("save called with options", options);
    console.log("pageCache", pageCache, chunkCache);
    let user = document.getElementById("username").innerText;
    if ((!user) || user === "") {
        let msg = "Username unset!";
        alert(msg);
        setEnabled(false);
        logError(msg);
        return;
    }
    if (options.status && (!pageCache || !pageCache.page.id)) {
        let msg = "No cached page -- cannot save!";
        alert(msg);
        setEnabled(false);
        logError(msg);
        return;
    }

    let payload = computeCurrentAnnotation(options, user);

    console.log("payload", payload);

    let request = {
        'client_id': clientID,
        'message_type': 'save',
        'payload': JSON.stringify(payload),
    };
    ws.send(JSON.stringify(request));
    let oldCache = pageCache;
    pageCache = payload;
    pageCache.file_type = oldCache.file_type;
    pageCache.offset = oldCache.offset;

    //updateStatusDisplay("page", payload.current_status);

    //console.log("old cache", oldCache);
    //console.log("saving new annotation", payload);
    //console.log("new cache", pageCache);
}


function saveUnlockAndNext(options) {
    lockGUI();
    console.log("saveUnlockAndNext called with options", options);
    let user = document.getElementById("username").innerText;
    if ((!user) || user === "") {
        let msg = "Username unset!";
        alert(msg);
        setEnabled(false);
        logError(msg);
        return;
    }
    if (options.status && (!pageCache || !pageCache.page.id)) {
        let msg = "No cached page -- cannot save!";
        alert(msg);
        setEnabled(false);
        logError(msg);
        return;
    }
    let unlock = {}
    if (pageCache && pageCache.page.id)
        unlock = { user_name: user, page_id: pageCache.page.id };

    let annotation = {};
    if (options.status) { // create annotation to save
        annotation = computeCurrentAnnotation(options, user);
    }

    let query = createQuery(options.stepSize, options.requestIndex, options.requestStatus, options.requestSource, options.requestAudioFile, options.requestTransRE);
    let payload = {
        annotation: annotation,
        unlock: unlock,
        query: query,
    };


    console.log("payload", payload);

    let request = {
        'client_id': clientID,
        'message_type': 'saveunlockandnext',
        'payload': JSON.stringify(payload),
    };
    console.log("saveunlockandnext sending payload", payload);
    ws.send(JSON.stringify(request));
}


onload = function () {

    //localStorage.clear();
    
    let params = new URLSearchParams(window.location.search);
    console.log("gloptions", gloptions);
    console.log("localStorage", localStorage);
    console.log("URL params", params);
    
    if (params.get('help') || params.get('options')) {
        let wrapper = document.getElementById("body");
        let options = ["username", "set_status", "request_status", "request_source", "request_transre", "request_index", "autoload"]; //, "context"];
        let html = "<h2>Available options</h2><ul>";
        for (let i = 0; i < options.length; i++)
            html = html + "<li>" + options[i] + "</li>";
        html = html + "</ul>"
        wrapper.innerHTML = html;
        return;
    }

    setEnabled(false);
    lockGUI();
    clear();
    document.getElementById("unlock-all").disabled = true;
    document.getElementById("unlock-all").classList.add("disabled");

    // if (params.get('context')) {
    //     gloptions.context = params.get('context');
    //     document.getElementById("context").innerText = `${gloptions.context} ms`;
    //     document.getElementById("context-view").classList.remove("hidden");
    // }
    // let cachedAutoplay = getFromURLParamsOrLocalStorage('autoplay', params)
    // if (cachedAutoplay === "false")
    // 	cachedAutoplay = false;
    // if (cachedAutoplay !== null && cachedAutoplay !== undefined) {
    //     document.getElementById("autoplay").checked = cachedAutoplay;
    // }
    let cachedAutoASR = getFromURLParamsOrLocalStorage('autoasr', params)
    if (cachedAutoASR === "false")
	cachedAutoASR = false;
    if (cachedAutoASR !== null && cachedAutoASR !== undefined) {
        document.getElementById("autoasr").checked = cachedAutoASR;
    }
    let cachedAutoplayOnASR = getFromURLParamsOrLocalStorage('autoplayonasr', params)
    if (cachedAutoplayOnASR === "false")
	cachedAutoplayOnASR = false;
    if (cachedAutoplayOnASR !== null && cachedAutoplayOnASR !== undefined) {
        document.getElementById("autoplayonasr").checked = cachedAutoplayOnASR;
    }

    let cachedSetStatus = getFromURLParamsOrLocalStorage('set_status', params);
    if (cachedSetStatus) {
	let ele = document.getElementById("setstatus");
        let options = ele.options;
        let seenSetStatus = false;
        for (let i = 0; i < options.length; i++) {
            if (options[i].value === ele.value) {
                ele.value = cachedSetStatus;
                seenSetStatus = true;
		break;
            }
        }
        if (!seenSetStatus) {
            logError(`Invalid set status: ${cachedSetStatus}`);
            ele.value = gloptions.defaultSetStatus;
        }
	localStorage.setItem('set_status', ele.value);
	if (ele.value !== gloptions.defaultSetStatus) {
	    ele.classList.add("search_active");
	}
    }
    
    let cachedRequestStatus = getFromURLParamsOrLocalStorage('request_status', params)
    if (cachedRequestStatus) {
	let ele = document.getElementById("requeststatus");
        let options = ele.options;
        let seenRequestedStatus = false;
        for (let i = 0; i < options.length; i++) {
            if (options[i].value === cachedRequestStatus) {
                ele.value = cachedRequestStatus;
                seenRequestedStatus = true;
		break;
            }
        }
        if (!seenRequestedStatus) {
            logError(`Invalid query status: $o{cachedRequestStatus}`);
            ele.value = gloptions.defaultRequestStatus;
        }
	localStorage.setItem('request_status', ele.value);
	if (ele.value !== gloptions.defaultRequestStatus) {
	    ele.classList.add("search_active");
	}
    }

    let cachedRequestSource = getFromURLParamsOrLocalStorage('request_source', params)
    if (cachedRequestSource) {
	let ele = document.getElementById("requestsource");
        let options = ele.options;
        let seenRequestedSource = false;
        for (let i = 0; i < options.length; i++) {
            if (options[i].value === cachedRequestSource) {
                ele.value = cachedRequestSource;
                seenRequestedSource = true;
		break;
            }
        }
        if (!seenRequestedSource) {
            logError(`Invalid query source: ${cachedRequestSource}`);
            ele.value = gloptions.defaultRequestSource;
        }
	localStorage.setItem('request_source', ele.value);
	if (ele.value !== gloptions.defaultRequestSource) {
	    ele.classList.add("search_active");
	}
    }

    if (params.get('request_transre')) {
        document.getElementById("requesttransre").value = params.get('request_transre');
    }

    let requestIndex;
    if (params.get('request_index')) {
        requestIndex = parseInt(params.get('request_index').toLowerCase()) - 1;
        requestIndex = requestIndex + "";
    }

    if (params.get('username')) {
	document.getElementById("username").innerText = params.get("username").toLowerCase();
    }
    else {
        let suggest = localStorage.getItem("username");
        if (!suggest || suggest === null)
            suggest = "";
        let username = prompt("User name", suggest);
        if (!username || username === null || username.trim() === "") {
            let msg = "Username unset!";
            logError(msg);
            alert(msg);
            return;
        }
	document.getElementById("username").innerText = username.toLowerCase();
    }
    localStorage.setItem("username", document.getElementById("username").innerText);


    let url = wsBase + "/ws/" + clientID + "/" + document.getElementById("username").innerText;
    ws = new WebSocket(url);
    ws.onopen = function () {
        logMessage("Websocket opened");
        if (requestIndex)
            saveUnlockAndNext({ requestIndex: requestIndex });
        else if (params.get('autoload'))
            saveUnlockAndNext({ stepSize: 1, requestStatus: document.getElementById("requeststatus").value});
        // else
        //     saveUnlockAndNext({ stepSize: 1, requestStatus: document.getElementById("requeststatus").value});
        setEnabled(false);
        document.getElementById("load_stats").click();
        listAvailableAudioFiles();
    }
    ws.onclose = function () {
        let msg = "Connection was closed from server";
        logError(msg);
        clear();
        setEnabled(false);
        enableStart(false);
        document.getElementById("unlock-all").disabled = true;
        document.getElementById("unlock-all").classList.add("disabled");

        ws = undefined;
        alert(msg);
    }
    ws.onerror = function (evt) {
        console.log("Websocket error", evt);
        let msg = "Websocket error";
        logError(msg);
        clear();
        setEnabled(false);
        enableStart(false);
        document.getElementById("unlock-all").disabled = true;
        document.getElementById("unlock-all").classList.add("disabled");

        ws = undefined;
        alert(msg);
    }
    ws.onmessage = function (evt) {
        let resp = JSON.parse(evt.data);
        //console.log("ws.onmessage", resp);
        if (resp.fatal) {
            ws.close();
            logError("Non-recoverable server error: " + resp.fatal);
            clear();
            setEnabled(false);
            enableStart(false);
            document.getElementById("unlock-all").disabled = true;
            document.getElementById("unlock-all").classList.add("disabled");

            ws = undefined;
            alert("Non-recoverable server error: " + resp.fatal);
            return;
        }
        if (resp.error) {
            logError("Server error: " + resp.error);
            alert("Server error: " + resp.error);
            return;
        }
        if (resp.info) {
            logMessage(resp.info);
        }
        if (resp.message_type === "project_name")
            document.getElementById("project_name").innerHTML = ": " + JSON.parse(resp.payload);
        else if (resp.message_type === "stats")
            displayStats(JSON.parse(resp.payload));
        else if (resp.message_type === "explicit_unlock_completed") {
            pageCache = null;
            chunkCache = null;
            logMessage(JSON.parse(resp.payload));
        }
        else if (resp.message_type === "no_audio_chunk") {
            let msg = JSON.parse(resp.payload);
            logMessage(msg);
            if (pageCache && pageCache !== null)
                setEnabled(true);
            else
                setEnabled(false);
            enableStart(true);
            alert(msg);
        }
        else if (resp.message_type === "audio_chunk") {
            displayAnnotationWithAudioData(JSON.parse(resp.payload));
        }
        else if (resp.message_type === "list-db-audio-files-response") {
            let select = document.getElementById("requestaudiofile");
            select.innerHTML = "";
            const files = JSON.parse(resp.payload);
	    //HB
	    console.log(files)
            for (let i = 0; i < files.length; i++) {
                let file = files[i];
                let option = document.createElement("option");
                option.value = file;
                option.innerText = file;
                select.appendChild(option);
            }
            let option = document.createElement("option");
            option.value = "any";
            option.innerText = "any";
            option.selected = "selected";
            select.appendChild(option);
	    
	    let cached = getFromURLParamsOrLocalStorage("request_audio_file");
            if (cached) {
		let ele = document.getElementById("requestaudiofile");
                let options = ele.options;
                let seenRequestedAudioFile = false;
                for (let i = 0; i < options.length; i++) {
                    if (options[i].value === cached) {
                        ele.value = cached;
                        seenRequestedAudioFile = true;
			break;
                    }
                }
                if (!seenRequestedAudioFile) {
                    logError(`Invalid query audiofile: ${cached}`);
                    ele.value = gloptions.defaultRequestAudioFile;
                }
		if (ele.value !== gloptions.defaultRequestAudioFile) {
		    ele.classList.add("search_active");
		}
		ele.title = "Audio: " + ele.value;
                localStorage.setItem("request_audio_file", ele.value);
		document.getElementById("load_stats").click();
            }
        }
        else if (resp.message_type === "asr-response") {
            document.getElementById("asr_info").innerText = "";
            document.getElementById("reverse_expansion").innerText = "";
            let asr = JSON.parse(resp.payload);
            // only update the text area if the ids match
            if (waveform.getSelectedRegion() && asr.uuid === waveform.getSelectedRegion().uuid) {
                if (asr.text === "") {
                    let asrInfo = document.getElementById("asr_info");
                    asrInfo.innerText = "no asr output";
                    let newAsrInfo = asrInfo.cloneNode(true);
                    asrInfo.parentNode.replaceChild(newAsrInfo, asrInfo);
                }
                else {
                    document.getElementById("editor-text-area").innerText = asr.text;
                    document.getElementById("editor-text-area").focus();
                    cacheActiveTranscription();
                }
                if (document.getElementById("autoplayonasr").checked)
                    document.getElementById("play-selected").click();
            }
        }
        else if (resp.info === "" && resp.message_type !== "keep_alive")
            logWarning("Unknown message from server: [" + resp.message_type + "] " + resp.payload);
    }

    console.log("main window loaded");

    let options = {
        waveformElementID: "waveform",
        timelineElementID: "waveform-timeline",
        // spectrogramElementID: "waveform-spectrogram",
        zoomElementID: "waveform-zoom",
        //navigationElementID: "waveform-navigation",
        debug: false,
        // regionMaxLength: 30,
        // regionMinLength: 0.1,
    };

    waveform = new Waveform(options);
    waveform.onSelectedRegionChange = onSelectedRegionChange;
    waveform.onUserAddedRegion = onUserAddedRegion;
    waveform.autoplayEnabledFunc = autoplayEnabledFunc;
    // waveform.wavesurfer.on("region-click", function (evt) {
    //     console.log("app.js region-click", evt.target);
    // });

    loadKeyboardShortcuts();
};

function autoplayEnabledFunc() {    
    //console.log("autoplayEnabledFunc called");
    return (document.getElementById("autoplay").checked && !document.getElementById("autoasr").checked);
    // if (document.getElementById("autoplay").checked && !document.getElementById("autoasr").checked)
    // 	document.getElementById("play-selected").click();
}

function loadKeyboardShortcuts() {
    let ele = document.getElementById("shortcuts");
    ele.innerHTML = "";
    Object.keys(shortcuts).forEach(function (key) {
        let id = shortcuts[key].buttonID;
        let tooltip = shortcuts[key].tooltip;
        if (!tooltip)
            tooltip = key.toLowerCase();
        if (id && tooltip) {
            let ele = document.getElementById(id);
            if (ele) {
                if (!ele.title) {
                    if (shortcuts[key].funcDesc)
                        ele.title = shortcuts[key].funcDesc + " - key: " + tooltip;
                }
            } else
                throw Error(`No element with id ${id}`);
        }
        if (tooltip && shortcuts[key].funcDesc) {
            let tr = document.createElement("tr");
            let td1 = document.createElement("td");
            let td2 = document.createElement("td");
            td1.innerHTML = tooltip;
            td2.innerHTML = shortcuts[key].funcDesc;
            tr.appendChild(td1);
            tr.appendChild(td2);
            ele.appendChild(tr);
        }
    });
}

function deleteSelectedChunk() {
    let regions = waveform.listRegions();
    for (let id in regions) {
        let region = regions[id];
        if (region.element.classList.contains("selected")) {
            region.remove();
        }
        let chunk = waveform.region2chunk(region);
        document.getElementById("editor-text-area").innerText = "";
        document.getElementById("editor-text-area").setAttribute("readonly", "readonly");
    }
}

const shortcuts = {
    // 'ctrl ArrowLeft': { funcDesc: `Move left boundary ${gloptions.boundaryMovementShort} ms to the left`, buttonID: 'move-left2left-short' },
    // 'ctrl ArrowRight': { funcDesc: `Move left boundary ${gloptions.boundaryMovementShort} ms to the right`, buttonID: 'move-left2right-short' },
    // 'shift ArrowLeft': { funcDesc: `Move right boundary ${gloptions.boundaryMovementShort} ms to the left`, buttonID: 'move-right2left-short' },
    // 'shift ArrowRight': { funcDesc: `Move right boundary ${gloptions.boundaryMovementShort} ms to the right`, buttonID: 'move-right2right-short' },

    // 'ctrl ArrowUp': { funcDesc: `Move left boundary ${gloptions.boundaryMovementLong} ms to the left`, buttonID: 'move-left2left-long' },
    // 'ctrl ArrowDown': { funcDesc: `Move left boundary ${gloptions.boundaryMovementLong} ms to the right`, buttonID: 'move-left2right-long' },
    // 'shift ArrowUp': { funcDesc: `Move right boundary ${gloptions.boundaryMovementLong} ms to the left`, buttonID: 'move-right2left-long' },
    // 'shift ArrowDown': { funcDesc: `Move right boundary ${gloptions.boundaryMovementLong} ms to the right`, buttonID: 'move-right2right-long' },

    'ctrl Enter': { funcDesc: 'Play selected chunk', buttonID: 'play-selected' },
    'ctrl alt Enter': { funcDesc: 'ASR selected chunk (and play on result)', buttonID: 'asr-request' },
    //'shift  ': { buttonID: 'play-selected' }, // hidden from shortcut view
    'ctrl  ': { tooltip: 'ctrl space', funcDesc: 'Play/pause waveform (playback will start at the cursor)', buttonID: 'waveform-playpause' },
    'alt o': { funcDesc: 'Save chunk as ok and go to next', buttonID: 'save-ok-next' },
    'alt s': { funcDesc: 'Save chunk as skip and go to next', buttonID: 'save-skip-next' },
    'ctrl ArrowDown': { funcDesc: 'Select next chunk', buttonID: 'waveform-skipforward' },
    'ctrl ArrowUp': { funcDesc: 'Select previous chunk', buttonID: 'waveform-skipback' },
    'ctrl Home': { funcDesc: 'Select first chunk', buttonID: 'waveform-skiptofirst' },
    'ctrl End': { funcDesc: 'Select final chunk', buttonID: 'waveform-skiptolast' },
    // 'ctrl alt ArrowDown': { funcDesc: 'Go to next page', buttonID: 'next_page_any' },
    // 'ctrl alt ArrowUp': { funcDesc: 'Go to previous page', buttonID: 'prev_page_any' },
    //'ctrl alt ArrowDown': { funcDesc: 'Go to next page matching query request', buttonID: 'next_page' },
    //'ctrl alt ArrowUp': { funcDesc: 'Go to previous page matching query request', buttonID: 'prev_page' },
    'ctrl Delete': { funcDesc: 'Delete selected chunk', // func: deleteSelectedChunk, 
		     buttonID: 'delete-selected' }
};

window.addEventListener("keydown", function (evt) {
    // if (document.activeElement.tagName.toLowerCase() === "textarea")
    //     return;
    let key = evt.key;
    if (evt.altKey)
        key = "alt " + key;
    if (evt.ctrlKey)
        key = "ctrl " + key;
    if (evt.shiftKey)
        key = "shift " + key;
    //console.log(evt.key, evt.keyCode, evt.ctrlKey, evt.altKey, "=>", key);
    if (shortcuts[key]) {
        evt.preventDefault();
        let shortcut = shortcuts[key];
        if ((!shortcut.alt && !evt.altKey) || (!shortcut.ctrl && !evt.ctrlKey) || (!shortcut.shift && !evt.shiftKey) ||
            (shortcut.ctrl && evt.ctrlKey) || (shortcut.alt && evt.altKey) || (shortcut.shift && evt.shiftKey)) {
            if (shortcut.buttonID) {
                document.getElementById(shortcut.buttonID).click();
            } else if (shortcut.func) {
                shortcut.func();
            }
            return false;
        }
    }
});

window.onbeforeunload = function () {
    if (pageCache && pageCache != null) {
        return "Are you sure you want to navigate away?";
    }
}

function time_convert(ms) {
    var milliseconds = parseInt((ms % 1000) / 100),
        seconds = Math.floor((ms / 1000) % 60),
        minutes = Math.floor((ms / (1000 * 60)) % 60),
        hours = Math.floor((ms / (1000 * 60 * 60)) % 24);

    //hours = (hours < 10) ? "0" + hours : hours;
    minutes = minutes + (hours * 60);
    minutes = (minutes < 10) ? "0" + minutes : minutes;
    seconds = (seconds < 10) ? "0" + seconds : seconds;

    if (milliseconds < 100)
        milliseconds = "00" + milliseconds;
    else if (milliseconds < 10)
        milliseconds = "0" + milliseconds;

    //return hours + ":" + minutes + ":" + seconds + "." + milliseconds;
    return minutes + ":" + seconds + "." + milliseconds;
}

