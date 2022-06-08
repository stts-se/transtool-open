'use strict';

document.getElementById("editor-text-area").addEventListener("keyup", function (evt) {
    //console.log("EVT", evt);
    cacheActiveTranscription();
});

function cacheActiveTranscription(setStatus) {
    //console.log("cacheTranscription");

    var text = document.getElementById("editor-text-area").innerText;
    text = text.trim();
    text = text.replace(/[\s]{2,}/g, " ");

    // if (text.trim() === "") {
    //     return;
    // };

    // console.log("save_trans.js text: <" + text + ">");
    
    let selected = waveform.getSelectedRegion();
    if (selected) {
        let ch = chunkCache[selected.uuid];
        if (ch) {
           //console.log("chunk before", chunkCache[selected.uuid]);
            ch.trans = text;

            if (setStatus) {
                let statusHistory = ch.status_history;
                if (!statusHistory)
                    statusHistory = [];
                if (ch.current_status.name !== "unchecked")
                    statusHistory.push(ch.current_status);
                ch.current_status = setStatus;
            }

            //console.log("chunk after", chunkCache[selected.uuid]);
	    return ch;
        }
    }
}


