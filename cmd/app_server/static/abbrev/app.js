"use strict";

//const baseURL = window.location.protocol + '//' + window.location.host + window.location.pathname.replace(/\/$/g, "");

const keyCodeEnter = 13;
const keyCodeSpace = 32;
const keyCodeEscape = 27;

let abbrevMap = {};
let reverseAbbrevMap = {};

//let inputtext = document.getElementById("editortextarea");
let userName = document.getElementById("username");
//let sessionname = document.getElementById("sessionname");

// ------------------
// ABBREVS


async function loadAbbrevListNames(firstLoad) {
	await fetch(baseURL + "/abbrev/list_lists_with_length")
		.then(response => response.json())
		.then(jzon => populateAbbrevListNames(jzon, firstLoad));

}

function nonEmptyString(s) {
	return s !== undefined && s !== null && s.trim().length > 0;
}

function removeClass(element, theClass) {
	let allC = element.getAttribute("class");
	if (nonEmptyString(allC)) {
		let newCs = [];
		let allCs = allC.split(/ +/);
		for (let i = 0; i < allCs.length; i++) {
			const thisC = allCs[i];
			if (thisC !== theClass) {
				newCs.push(thisC);
			}
		}
		element.setAttribute("class", newCs.join(" "));
	}
}

function removeStyle(element, theStyle) {
	let allS = element.getAttribute("style");
	if (nonEmptyString(allS)) {
		let newSs = [];
		let allSs = allS.split(/ *; +/);
		for (let i = 0; i < allSs.length; i++) {
			const thisS = allSs[i];
			let key = thisS.split(/ *: */)[0].trim();
			if (key !== theStyle) {
				newSs.push(thisS);
			}
		}
		element.setAttribute("style", newSs.join(" "));
	}
}


function validateRegisterButton() {
	//console.log("validateRegisterButton","uName:", username.value, "sName:", sessionname.value);
	const valid = nonEmptyString(username.value) && nonEmptyString(sessionname.value);
	if (valid) {
		register.removeAttribute("disabled");
	} else {
		register.setAttribute("disabled", "disabled");
	}
}

function findIndexOfListName(listName) {
	let lists = listUserAbbrevListsWithInfo();
	for (let i = 0; i < lists.length; i++) {
		let list = lists[i];
		if (list.name == listName) {
			return i;
		}
	}
	return -1;
}

function enableDisableMoveButtons(row, index, totalCount) {
	let up = row.children[1].children[0]; // TODO: brittle
	let down = row.children[1].children[1]; // TODO: brittle
	if (index < 1) {
		up.setAttribute("disabled", "disabled");
		up.setAttribute("class", up.getAttribute("class") + " disabled");
	} else {
		up.removeAttribute("disabled");
		removeClass(up, "disabled");
	}
	if (index >= totalCount - 1) {
		down.setAttribute("disabled", "disabled");
		down.setAttribute("class", down.getAttribute("class") + " disabled");
	} else {
		down.removeAttribute("disabled");
		removeClass(down, "disabled");
	}

}

function saveAbbrevListSettingsToLocalStorage() {
	let lists = listUserAbbrevListsWithInfo();
	for (let i = 0; i < lists.length; i++) {
		lists[i].count = undefined;
	}
	console.log("saveAbbrevListSettingsToLocalStorage", lists);
	localStorage.setItem('abbrev_lists', JSON.stringify(lists));
}

function moveListUp(listName) {
	let oldIndex = findIndexOfListName(listName);
	let newIndex = oldIndex - 1;
	let tbody = document.getElementById("abbrev_list_names");
	let currRow = tbody.children[oldIndex];
	let swapRow = tbody.children[newIndex];
	currRow.parentNode.insertBefore(currRow, swapRow);
	enableDisableMoveButtons(currRow, newIndex, tbody.children.length);
	enableDisableMoveButtons(swapRow, oldIndex, tbody.children.length);
	loadSelectedAbbrevLists();
	saveAbbrevListSettingsToLocalStorage();
}
function moveListDown(listName) {
	let oldIndex = findIndexOfListName(listName);
	let newIndex = oldIndex + 1;
	let tbody = document.getElementById("abbrev_list_names");
	let currRow = tbody.children[oldIndex];
	let swapRow = tbody.children[newIndex];
	currRow.parentNode.insertBefore(swapRow, currRow);
	enableDisableMoveButtons(currRow, newIndex, tbody.children.length);
	enableDisableMoveButtons(swapRow, oldIndex, tbody.children.length);
	loadSelectedAbbrevLists();
	saveAbbrevListSettingsToLocalStorage();
}

// called multiple times if list names are updated from server (overwriting content)
function populateAbbrevListNames(listNamesJS, firstLoad) {
	let cached = listUserAbbrevListsWithInfo();
	if (cached.length === 0) {
		let localStorageOption = localStorage.getItem("abbrev_lists");
		//console.log("localstorage", localStorageOption);
		if (localStorageOption !== undefined && localStorageOption !== null)
			cached = JSON.parse(localStorageOption);
	}
	let oldSort = [];
	let oldSelected = [];
	cached.forEach(function(item) {
		//console.log(item);
		if (item.checked)
			oldSelected.push(item.name);
		oldSort.push(item.name);
	});
    console.log("populateAbbrevListNames", cached, oldSort, oldSelected);
    if (listNamesJS === null || listNamesJS === undefined)  {
	console.error("No abbreviation lists");
	return [];
    };
    listNamesJS = listNamesJS.sort(function (a, b) {
	return oldSort.indexOf(a.name) > oldSort.indexOf(b.name)
    });
    
	let tbody = document.getElementById("abbrev_list_names");
	tbody.innerHTML = '';

	let index = -1;
	let total = listNamesJS.length;
	listNamesJS.forEach(item => {
		index++;
		let tr = document.createElement("tr");

		let cb = document.createElement("input");
		cb.setAttribute("type", "checkbox");

		if (firstLoad && oldSort.length === 0) {
			cb.setAttribute("checked", "checked");
		} else if (oldSelected.includes(item.name)) {
			cb.setAttribute("checked", "checked");
		}
		cb.setAttribute("name", "abbrev_list");
		cb.setAttribute("value", item.name);
		cb.setAttribute("count", item.length);
		cb.addEventListener("change", function () {
			let listName = this.getAttribute("value");
			loadSelectedAbbrevLists();
			saveAbbrevListSettingsToLocalStorage();
		});

		let text = document.createElement("span");
		text.innerHTML = item.name + " (" + item.length + ")";
		let label = document.createElement("label");
		label.appendChild(cb);
		label.appendChild(text);

		let td1 = document.createElement("td");
		td1.appendChild(label);

		let move = document.createElement("td");
		let up = document.createElement("button");
		up.listName = item.name;
		up.setAttribute("title", "Move up");
		up.setAttribute("class", "btn noborder move up");
		up.innerHTML = "&#8593;";
		up.addEventListener("click", function (evt) {
			let name = evt.target.listName;
			moveListUp(name);
		});
		// if (index < 1) {
		//     up.setAttribute("disabled", "disabled");
		//     up.setAttribute("class", up.getAttribute("class") + " disabled");
		// }

		let down = document.createElement("button");
		down.listName = item.name;
		down.setAttribute("class", "btn noborder move down");
		down.innerHTML = "&#8595;";
		down.title = "Move down";
		down.style["display"] = "none";
		down.addEventListener("click", function (evt) {
			let name = evt.target.listName;
			moveListDown(name);
		});
		// if (index >= total-1) {	    
		//     down.setAttribute("disabled", "disabled");
		//     down.setAttribute("class", down.getAttribute("class") + " disabled");
		// }

		move.appendChild(up);
		move.appendChild(down);

		tr.appendChild(td1);
		tr.appendChild(move);

		enableDisableMoveButtons(tr, index, total);
		tbody.appendChild(tr);
	});

	console.log("Updated abbrev list names");
	loadSelectedAbbrevLists();

}

function listSelectedAbbrevLists() {
	let res = [];
	let lists = listUserAbbrevListsWithInfo();
	for (let i = 0; i < lists.length; i++) {
		let list = lists[i];
		if (list.checked)
			res.push(list.name);
	}
	return res;
}

function listUserSortedAbbrevLists() {
	let res = [];
	let lists = listUserAbbrevListsWithInfo();
	for (let i = 0; i < lists.length; i++) {
		let list = lists[i];
		res.push(list.name);
	}
	return res;
}

function listUserAbbrevListsWithInfo() {
	let res = [];
	let rows = document.getElementById("abbrev_list_names").children;
	for (let i = 0; i < rows.length; i++) {
		let td = rows[i].children[0];
		let cb = td.children[0].children[0]; // TODO brittle
		let checked = cb.checked;
		let list = {};
		list.checked = checked;
		list.name = cb.value;
		list.count = cb.length;
		res.push(list);
	}
	//console.log("listUserAbbrevLists", res);
	return res;
}

function loadSelectedAbbrevLists() {
	let listNames = listSelectedAbbrevLists();
	loadAbbrevs(listNames);
}

async function loadAbbrevs(listNames) {
	let newMap = {};
	let hasError = false; // TODO: Does this work as intended?
	for (let li = 0; li < listNames.length; li++) {
		const listName = listNames[li];
		console.log("Loading " + listName); //  + " " + new Date())
		await fetch(baseURL + "/abbrev/list_abbrevs/" + listName).then(async function (r) {
			if (r.ok) {
				const serverAbbrevs = await r.json();
				for (let i = 0; i < serverAbbrevs.length; i++) {
					//console.log("i: ", i, serverAbbrevs[i]);
					const a = serverAbbrevs[i];
					if (!newMap.hasOwnProperty(a.abbrev)) {
						newMap[a.abbrev] = a.expansion;
					} else {
						//console.log("SKIPPING " + a.abbrev + " => " + a.expansion + ", ALREADY HAD " + newMap[a.abbrev]);
					}
					//console.log(a.abbrev, a.expansion);
				};
				console.log("Loaded " + listName + " (" + serverAbbrevs.length + ")"); // + new Date())
			} else {
				hasError = true;
				logAbbrevMessage("error", "failed to list abbreviations");
			}
		});
	}
	if (!hasError) {
		abbrevMap = newMap;
		updateReverseAbbrevMap();
	}
    console.log("Lists loaded: " + listNames + " (" + Object.keys(abbrevMap).length + " abbrevs)");
};

function updateReverseAbbrevMap() {
	Object.keys(abbrevMap).forEach(function (abbrev) {
		var word = abbrevMap[abbrev]
		if (!reverseAbbrevMap.hasOwnProperty(word)) {
			reverseAbbrevMap[word] = abbrev;
		}
	});
}

const leftWordRE = /(?:^|[ \n\r\t]+)([^ \n\r\t]+)$/; // TODO Really no need for regexp,
// just pick off characters until
// space, etc, or end of string?


//TODO Should be websocket
// function sendText(evt) {

//     let text = document.getElementById("editortextarea").value;


//     // TODO error handling
//     fetch(baseURL + "/incoming_text/" + encodeURIComponent(text));

// }


// function sendTextWS(evt) {
// 	if (producerWS !== null && producerWS !== undefined) {
// 		let text = document.getElementById("editortextarea").innerHTML;
// 		let js = {
// 			'label': 'text',
// 			'content': text,
// 		};
// 		producerWS.send(JSON.stringify(js));
// 		// TODO: Check that the send is OK (disconnected errors, etc)
// 	}
// }

let shouldTriggerExpansion = function (evt) {
	return (evt.key === " ")
}

function checkForExistingExpansion(evt) {
	setTimeout(function () {
		if (!evt.ctrlKey && (evt.key === " " || evt.key === "Enter")) {
			const word = wordLeftOfCursorAndSpace().trim();
			if (word === "") {
				return;
			};
			if (reverseAbbrevMap.hasOwnProperty(word)) {

				let revexp = document.getElementById("reverse_expansion");
				revexp.innerText = word + ":  " + reverseAbbrevMap[word];

				let newRevexp = revexp.cloneNode(true);
				revexp.parentNode.replaceChild(newRevexp, revexp);

				//revexp.setAttribute("class", "wiggle");
				//revexp.removeAttribute("class");
				//revexp.setAttribute("class", "wiggle");
				logAbbrevMessage("info", "Abbrev exists for " + word + " : " + reverseAbbrevMap[word]);
			}
		}
	});
}

function checkForAbbrev(evt) {
	setTimeout(function () {
		//console.log("evt", evt);
		const sel = window.getSelection();
		if (shouldTriggerExpansion(evt)) {
			// TODO: Call wordLeftOfCursorAndSpace
			const startPos = window.getSelection().getRangeAt(0).startOffset;
			const node = sel.focusNode;
			const text = node.textContent;

			const stringUp2Cursor = text.substring(0, startPos - 1);

			// wordBeforeSpace will have a trailing space
			const regexRes = leftWordRE.exec(stringUp2Cursor);
			if (regexRes === null) {
				return;
			};
			const wordBeforeSpace = regexRes[1];

			if (abbrevMap.hasOwnProperty(wordBeforeSpace.trim())) {
				// console.log(wordBeforeSpace, abbrevMap[wordBeforeSpace.trim()]);

				// Match found. Replace abbreviation with its expansion
				const textBefore = text.substring(0, startPos - wordBeforeSpace.length - 1);
				const textAfter = text.substring(startPos);
				const expansion = abbrevMap[wordBeforeSpace.trim()];

				// if first word in text, special treatment
				if (text.length === wordBeforeSpace.length + 1) { // + 1 = SPACE
				    node.data = expansion + "\u00A0" + textAfter;
				    //node.data = expansion + " " + textAfter;
				}
				else {
				    node.data = textBefore + expansion + "\u00A0" + textAfter;
				    //node.data = textBefore + expansion + " " + textAfter;
				};
				// Move cursor to directly after expanded word + 1 (space)
				const newOffset = (textBefore + " " + expansion).length;
				var range = document.createRange();
				range.setStart(node, newOffset);
				range.collapse(true);
				sel.removeAllRanges();
				sel.addRange(range);
				// TODO could be handled by the textarea, "on update" or similar?
				//sendTextWS();
			};
		}

	}, 0);

}

function wordLeftOfCursorAndSpace() {
	const sel = window.getSelection();

	const startPos = window.getSelection().getRangeAt(0).startOffset;
	const node = sel.focusNode;
	const text = node.textContent;
	const stringUp2Cursor = text.substring(0, startPos - 1);
	const regexRes = leftWordRE.exec(stringUp2Cursor);
	if (regexRes === null) {
		return "";
	};
	const wordBeforeSpace = regexRes[1];
	return wordBeforeSpace;
}

function wordLeftOfCursor() {
	const sel = window.getSelection();

	const startPos = window.getSelection().getRangeAt(0).startOffset;
	const node = sel.focusNode;
	const text = node.textContent;
	const stringUp2Cursor = text.substring(0, startPos);
	const regexRes = leftWordRE.exec(stringUp2Cursor);
	if (regexRes === null) {
		return "";
	};
	const wordBeforeSpace = regexRes[1];
	return wordBeforeSpace;
}


// document.getElementById("fullscreen").addEventListener("click", toggleFullscreen);

// function toggleFullscreen() {
//     console.log(document.getElementById("editortextarea"));
// }

// TODO
//document.getElementById("add_abbrev").addEventListener("click", abbrevPopUp);

function addNewAbbrevCreateListIfNotExists(listName, newAbbrev, newExpansion) {
    let a  = newAbbrev;
    let e  = newExpansion;

    
    if (!listName.match(/^[a-zåäöA-ZÅÄÖ0-9_ -]+$/)) {
	logMessage("error", "Invalid list name '" + e + "' (valid characters: 'a-zåäö0-9_ -')");
	return;
    }
      
    if (a === "") {
	logMessage("error", "Empty abbrev");
	return;
    }
    if (!a.match(/^[a-zåäöæøA-ZÅÄÖÆØ0-9_?:-]+$/)) {
	logMessage("error", "Invalid abbrev '" + a + "' (valid characters: 'a-zåäöæøA-ZÅÄÖÆØ0-9_?:-')");
	return;
    }
    if (!e.match(/^[a-zåäöA-ZÅÄÖ0-9_\]\[ ?:-]+$/)) {
	logMessage("error", "Invalid expansion '" + e + "' (valid characters: 'a-zåäöA-ZÅÄÖ0-9_ []:?-')");
	return;
    }
    
    // var selectedList = document.getElementById("selected_list_name").value;
    // // var listNames = document.getElementById("abbrev_list_names").children;
    // // for (var i = 0; i < listNames.length; i++) {

    // // 	console.log("??",listNames[i].children[0].children[0]);
    // // 	// TODO Very brittle. Make better way of getting radio elem, etc
    // // 	if (listNames[i].children[0].children[0].type === 'radio' && listNames[i].children[0].children[0].checked) {
    // //         // get value, set checked flag or do whatever you need to
    // //         selectedList = listNames[i].children[0].children[0].value.trim();       
    // // 	}
    // // }
    
    // if (selectedList === "") {
    // 	// TODO: error message
    // 	console.log("Cannot add new abbrev before selecting a list");
    // 	return;
    // }
    
    fetch(baseURL + "/abbrev/add_create_list_if_not_exists/" + listName + "/" + a + "/" + e).then(async function(response) {
	if (response.ok) {
	    // TODO: refactor with chain of then's 
	    let jzon = await response.json();
	    if (jzon.label === "error") {
	    	logMessage("error", jzon.content);
	    	return;
	    } else if (jzon.label === "info") {
	    	logMessage("info", jzon.content);
		// TODO Very brittle if table layout is changed!
		// let tr = document.createElement("tr");
		// let td1 = document.createElement("td");
		// td1.style["width"] = "50%";
		// let td2 = document.createElement("td");
		// td2.style["width"] = "50%";
		// let td3 = document.createElement("td");
		
		// td1.innerText = a;
		// td2.innerText = e;
		
		// td3.innerText = "×";
		// td3.setAttribute("value", a);
		// td3.setAttribute("list_name", selectedList);
		// td3.setAttribute("class", "red btn noborder");
		// //TODO Cutandpaste from other function
		// // Create a stand-alone function
		// td3.addEventListener("click", function() {
		    
		    
		//     // Name of abbrev list to delete
		//     let a = this.getAttribute("value");
		//     let ln = this.getAttribute("list_name");
		    
		//     fetch(baseURL + "/abbrev/delete/"+ ln + "/" + a).then(function(response) {
		// 	if(response.ok) {
		// 	    let abbrevs = document.getElementById("abbrevs_table").children;
		// 	    for (var i = 0; i < abbrevs.length; i++) {
		// 		let abr = abbrevs[i].children[0];
		// 		if (abr.innerText.trim() === a) {
		// 		    abbrevs[i].remove();
		// 		    document.getElementById("messages").innerText = "info :" + "Deleted '"+ a +"'";
		// 		} 
		// 	    }
		// 	    // Reload abbrevs after delete?
		// 	    // Or remove locally?
		// 	    loadAbbrevMap(ln);
		// 	}
		//     });
		    
	    //});
		
		
		
		// tr.appendChild(td1);
		// tr.appendChild(td2);
		// tr.appendChild(td3);
		
		// document.getElementById("abbrevs_table").prepend(tr)
		// let n = document.getElementById("abbrevs_table").children.length;
		// setAbbrevCountInListTable(selectedList, n);
		// //logMessage("info", "Added abbrevation " + a + " => " + e);
	    }
	} else {
	    logMessage("error", "Failed to add abbrevation " + a + " => " + e);
	}
    });
}




function abbrevPopUp(evt) {
	if (evt.key === "F9" || evt.target.id === "add_abbrev") {
		const word = wordLeftOfCursor().trim();
		if (word === "") {
			return;
		};
		let abbrev = prompt(word, "");
		if (abbrev !== undefined && abbrev !== null) {
			abbrev = abbrev.trim();
			if (abbrev !== "") {
				abbrevMap[abbrev] = word;
				reverseAbbrevMap[word] = abbrev;
			    logAbbrevMessage("info", "Added abbreviation to local storage: " + abbrev + " => " + word);
			    addNewAbbrevCreateListIfNotExists(document.getElementById("username").innerText, abbrev, word);
			}
		};
	}
}


function logAbbrevMessage(title, text, stacktrace) {
	if (stacktrace !== undefined) {
		//const stack = new Error().stack;
		console.log("logAbbrevMessage", title, text, stacktrace.stack);
	} else {
		console.log("logAbbrevMessage", title, text);
	}
	document.getElementById("messages").textContent = title + ": " + text;
}


//var producerWS; // TODO ws defined in ../app.js
//var register = document.getElementById("register");

// TODO remove code related to 'producer' role

// username.addEventListener("keypress", function (evt) {
// 	if (evt.keyCode == keyCodeEnter) {
// 		register.click();
// 	}
// });

// TODO
// sessionname.addEventListener("keypress", function (evt) {
// 	if (evt.keyCode == keyCodeEnter) {
// 		register.click();
// 	}
// });

//username.addEventListener("keyup", validateRegisterButton);
// TODO
//sessionname.addEventListener("keyup", validateRegisterButton);

// TODO
// register.addEventListener('click', function (evt) {
// 	let userName = username.value.trim();
// 	let sessionName = sessionname.value.trim();
// 	if (register.value === "Unregister") {
// 		let f = async function () {
// 			let url = baseURL + "/producer/unregister/" + userName + "/" + sessionName;
// 			await fetch(url).then(function (r) {
// 				if (r.ok) {

// 					let fc = async function () {
// 						let url = baseURL + "/consumer/unregister/" + userName + "/" + sessionName;
// 						await fetch(url).then(function (r) {
// 							if (r.ok) {
// 								//logAbbrevMessage("info", "unregistered consumer session " + userName + "/" + sessionName);
// 							} else {
// 								logAbbrevMessage("error", "couldn't unregister producer session " + userName + "/" + sessionName);
// 							}
// 						});
// 					}
// 					fc();

// 					logAbbrevMessage("info", "unregistered producer session " + userName + "/" + sessionName);
// 					register.value = "Register";
// 					username.removeAttribute("disabled");
// 					sessionname.removeAttribute("disabled");
// 					document.getElementById("add_abbrev").setAttribute("disabled", "disabled");
// 					inputtext.setAttribute("contenteditable", "false");
// 					inputtext.setAttribute("style", inputtext.getAttribute("style") + "; background: lightgrey");
// 					username.focus();
// 				} else {
// 					logAbbrevMessage("error", "couldn't unregister producer session " + userName + "/" + sessionName);
// 				}
// 			});
// 		}
// 		f();

// 	} else {

// 		if (!userName.match(/^[a-zåäö0-9_-]+$/)) {
// 			logAbbrevMessage("error", "Invalid user name '" + userName + "' (valid characters: 'a-zåäö0-9_-')");
// 			return;
// 		}
// 		if (!sessionName.match(/^[a-zåäö0-9_-]+$/)) {
// 			logAbbrevMessage("error", "Invalid session name '" + sessionName + "' (valid characters: 'a-zåäö0-9_-')");
// 			return;
// 		}

// 		let wsBase = baseURL.replace(/^http/, "ws");
// 		let prodWsURL = wsBase + "/ws/producer/register/" + userName + "/" + sessionName;
// 		producerWS = new WebSocket(prodWsURL);
// 		producerWS.onopen = function () {
// 			localStorage.setItem("username", userName);
// 		}
// 		producerWS.onerror = function () {
// 			logAbbrevMessage("error", "Couldn't register session " + sessionName);
// 		}
// 		producerWS.onmessage = function (evt) {
// 			let resp = JSON.parse(evt.data);
// 			if (resp.label === "registered_consumers") {
// 				populateRegisteredConsumers(resp.content.split(/, */));
// 			} else if (resp.label === "registered") {
// 				logAbbrevMessage("info", "Registered to session " + sessionName);
// 				username.setAttribute("disabled", "disabled");
// 				sessionname.setAttribute("disabled", "disabled");
// 				document.getElementById("add_abbrev").removeAttribute("disabled");
// 				inputtext.setAttribute("contenteditable", "true");
// 				inputtext.setAttribute("style", inputtext.getAttribute("style") + "; background: inherit");
// 				register.value = "Unregister";
// 				inputtext.focus();

// 				// register self-consumer
// 				let ws = new WebSocket(wsBase + "/ws/consumer/register/" + userName + "/" + sessionName);
// 				ws.onmessage = function (evt) {
// 					let resp = JSON.parse(evt.data);
// 					if (resp.label === "text") {
// 						if (resp.content !== undefined) {
// 							let output = document.getElementById("consumer_output");
// 							output.innerHTML = resp.content;
// 						}
// 					} else if (resp.label === "keepalive") {
// 						console.log("socket received keepalive");
// 					} else if (resp.label === "error") {
// 						logAbbrevMessage("error", resp.content);
// 					} else {
// 						logAbbrevMessage("info", resp.label + " - " + resp.content);
// 					}
// 				};

// 			} else if (resp.label === "error") {
// 				logAbbrevMessage("error", resp.content);
// 			} else if (resp.label === "keepalive") {
// 				console.log("socket received keepalive");
// 			} else {
// 				logAbbrevMessage("info", resp.label + " - " + resp.content);
// 			}
// 		}
// 	}

// });

// TODO
// document.getElementById("toggle_output_visible").addEventListener("click", function (evt) {
// 	let btn = evt.target;
// 	let outputArea = document.getElementById("consumer_output");
// 	if (btn.innerText.toLowerCase() === "hide") {
// 		outputArea.style["display"] = "none";
// 		btn.innerText = "Show";
// 	} else {
// 		outputArea.style["display"] = "";
// 		btn.innerText = "Hide";
// 	}
// });


function localStorageEnabled() {
	if (typeof localStorage !== 'undefined') {
		try {
			localStorage.setItem('feature_test', 'yes');
			if (localStorage.getItem('feature_test') === 'yes') {
				localStorage.removeItem('feature_test');
				return true;
				// localStorage is enabled
			} else {
				return false;
				// localStorage is disabled
			}
		} catch (e) {
			return false;
			// localStorage is disabled
		}
	} else {
		return false;
	}
}


function loadSettings() {
	if (localStorage.hasOwnProperty("color_scheme")) {
		let value = localStorage.getItem("color_scheme");
		setColorScheme(value);
		let options = document.getElementById("color_scheme_selection");
		for (let i = 0; i < options.length; i++) {
			let opt = options[i];
			opt.selected = (opt.value === value);
		}
	}
	if (localStorage.hasOwnProperty("output_font_size")) {
		let value = localStorage.getItem("output_font_size");
		document.getElementById("consumer_output").style.fontSize = value;
		let options = document.getElementById("output_font_size_selection");
		for (let i = 0; i < options.length; i++) {
			let opt = options[i];
			opt.selected = (opt.value === value);
		}
	}
	if (localStorage.hasOwnProperty("output_font_family")) {
		let value = localStorage.getItem("output_font_size");
		document.getElementById("consumer_output").style.fontFamily = value;
		let options = document.getElementById("output_font_family_selection");
		for (let i = 0; i < options.length; i++) {
			let opt = options[i];
			opt.selected = (opt.value === value);
		}
	}
	if (localStorage.hasOwnProperty("input_font_size")) {
		let value = localStorage.getItem("input_font_size");
		document.getElementById("editor-text-area").style.fontSize = value;
		let options = document.getElementById("input_font_size_selection");
		for (let i = 0; i < options.length; i++) {
			let opt = options[i];
			opt.selected = (opt.value === value);
		}
	}
	if (localStorage.hasOwnProperty("input_font_family")) {
		let value = localStorage.getItem("input_font_size");
		document.getElementById("editor-text-area").style.fontFamily = value;
		let options = document.getElementById("input_font_family_selection");
		for (let i = 0; i < options.length; i++) {
			let opt = options[i];
			opt.selected = (opt.value === value);
		}
	}
	if (localStorage.hasOwnProperty("expansion_trigger")) {
		let value = localStorage.getItem("expansion_trigger");
		setExpansionTrigger(value);
		let options = document.getElementById("expansion_trigger_selection");
		for (let i = 0; i < options.length; i++) {
			let opt = options[i];
			opt.selected = (opt.value === value);
		}
	}
}

let startMeUp = function () {

// 	if (!localStorageEnabled()) {
// 		alert("Your browser does not support localStorage.");
// 		return;
// 	}

// 	var urlParams = new URLSearchParams(window.location.search);
// 	if (urlParams.has('username')) {
// 		username.value = urlParams.get("username");
// 	} else if (localStorage.hasOwnProperty("username")) {
// 		username.value = localStorage.getItem("username");
// 	}
// 	if (urlParams.has('sessionname')) {
// 		sessionname.value = urlParams.get("sessionname");
// 	}
 	loadAbbrevListNames(true);
// 	loadSettings();
// 	//loadSelectedAbbrevLists();

// 	// textarea.addEventListener('keyup', sendTextWS); // TODO: Should listen to value changed, but there is no such event
// 	// textarea.addEventListener('keypress', sendTextWS); // TEST | HL ADDED 20191108
    // 	// textarea.addEventListener('keydown', sendTextWS); // TEST | HL ADDED 20191108
    document.getElementById("editor-text-area").addEventListener('keypress', checkForAbbrev);
    // 	// NB: This must be keydown, if keypress also just expanded abbreviations will be reported (which is pointless).
    document.getElementById("editor-text-area").addEventListener('keydown', checkForExistingExpansion);

// 	// username.focus();

// 	// let wsBase = baseURL.replace(/^http/, "ws");
// 	// let url = wsBase + "/ws/producer/pageloaded";
// 	// console.log("Producer websocket URL", url);
// 	// let ws0 = new WebSocket(url);
// 	// ws0.onopen = function () {
// 	// 	logAbbrevMessage("info", "Subscribed to global producer websocket");
// 	// }
// 	// ws0.onmessage = function (evt) {
// 	// 	let resp = JSON.parse(evt.data);
// 	// 	if (resp.label === "abbrevs_updated") {
// 	// 		loadAbbrevListNames(false);
// 	// 		loadSelectedAbbrevLists();
// 	// 		logAbbrevMessage("info", "abbrevs updated");
// 	// 	} else if (resp.label === "abbrev_lists_updated") {
// 	// 		loadAbbrevListNames(false);
// 	// 		logAbbrevMessage("info", "abbrev list names updated");
// 	// 	} else if (resp.label === "keepalive") {
// 	// 		console.log("socket received keepalive");
// 	// 	} else {
// 	// 		logAbbrevMessage("info", resp.label + " - " + resp.content);
// 	// 	}
// 	// };
// 	// ws0.onerror = function () {
// 	// 	logAbbrevMessage("error", "Couldn't subscribe to global consumer websocket");
// 	// }

//     // TODO
// 	// document.getElementById("input_font_size_selection").addEventListener("change", function (evt) {
// 	// 	let fontSize = this.value;
// 	// 	document.getElementById("editortextarea").style.fontSize = fontSize;
// 	// 	localStorage.setItem("input_font_size", fontSize);
// 	// });
// 	// document.getElementById("input_font_family_selection").addEventListener("change", function (evt) {
// 	// 	let fontFam = this.value;
// 	// 	document.getElementById("editortextarea").style.fontFamily = fontFam;
// 	// 	localStorage.setItem("input_font_family", fontFam);
// 	// });
// 	// document.getElementById("output_font_size_selection").addEventListener("change", function (evt) {
// 	// 	let fontSize = this.value;
// 	// 	document.getElementById("consumer_output").style.fontSize = fontSize;
// 	// 	localStorage.setItem("output_font_size", fontSize);
// 	// });
// 	// document.getElementById("output_font_family_selection").addEventListener("change", function (evt) {
// 	// 	let fontFam = this.value;
// 	// 	document.getElementById("consumer_output").style.fontFamily = fontFam;
// 	// 	localStorage.setItem("output_font_family", fontFam);
// 	// });
// 	// // document.getElementById("scale_selection").addEventListener("change", function(evt){	
// 	// // 	document.body.style.transform = "scale(" + this.value + ")";
// 	// // 	//document.body.style.zoom = this.value;
// 	// // });
// 	// document.getElementById("expansion_trigger_selection").addEventListener("change", function (evt) {
// 	// 	let trigger = this.value;
// 	// 	localStorage.setItem("expansion_trigger", trigger);
// 	// 	setExpansionTrigger(trigger);
// 	// });
// 	// document.getElementById("color_scheme_selection").addEventListener("change", function (evt) {
// 	// 	let scheme = this.value;
// 	// 	localStorage.setItem("color_scheme", scheme);
// 	// 	setColorScheme(scheme);
// 	// });


// 	// New abbrev pop-up
// 	// TODO Fill in pop-up for adding abbreviatio on the fly here.
// 	// TODO Currently, the word left of the cursor
 	window.addEventListener('keyup', abbrevPopUp);
 }

// TODO 
startMeUp();

function setExpansionTrigger(trigger) {
	// if (trigger === "ctrl+space") { // doesn't work
	//     shouldTriggerExpansion = function(evt) { console.log(evt.ctrlKey, evt.key); return (evt.ctrlKey && evt.key === " ") }
	// } else 
	if (trigger === "shift+space") {
		shouldTriggerExpansion = function (evt) { return (evt.shiftKey && evt.key === " ") }
	} else if (trigger === "space") {
		shouldTriggerExpansion = function (evt) { return (evt.key === " ") }
	} else if (trigger === "space-only") {
		shouldTriggerExpansion = function (evt) { return (evt.key === " " && !evt.shiftKey && !evt.ctrlKey && !evt.altKey) }
	} else {
		logAbbrevMessage("info", "Cannot set expansion trigger to " + trigger);
	}
}

function setColorScheme(scheme) {
	let bg = "";
	let fg = "";
	let lb = ""; // label (text/foreground) color
	if (scheme === "black") {
		bg = "black";
		fg = "#F0F0F0";
		lb = fg;
	} else if (scheme === "blue") {
		bg = "#000099";
		fg = "#F0F0F0";
		lb = fg;
	} else if (scheme === "gray") {
		bg = "#A9A9A9";
		fg = "#F0F0F0";
		lb = fg;
	} else if (scheme === "green") {
		bg = "green";
		fg = "#F0F0F0";
		lb = fg;
	} else if (scheme === "light") {
		lb = "#212529";
	} else {
		console.log("Unknown color scheme", scheme);
	}
	removeStyle(document.getElementById("body"), "background-color");
	removeStyle(document.getElementById("body"), "color");
	let labels = document.getElementsByClassName("label");
	for (let i = 0; i < labels.length; i++) {
		removeStyle(labels[i], "color");
	}
	if (bg !== "") {
		document.getElementById("body").style["background-color"] = bg;
	} if (fg !== "") {
		document.getElementById("body").style["color"] = fg;
	} if (lb !== "") {
		let labels = document.getElementsByClassName("label");
		for (let i = 0; i < labels.length; i++) {
			labels[i].style["color"] = fg;
		}
	}
}

// window.onbeforeunload = function () {
//     return "Are you sure you want to navigate away?";
// }



// OVERRIDE KEYBOARD EVENTS
// Chrome in "kiosk mode": google-chrome --app=http://localhost:3000

// window.onkeydown = overrideKeyboardEvent;
// window.onkeyup = overrideKeyboardEvent;
// let keyIsDown = {};

// let overrideKeyCodes = new Set([
//     // 9, // Tab
//     17, // Ctrl
//     18, // Alt

//     // F1-F12
//     112,
//     113,
//     114,
//     115,
//     116,
//     117,
//     118,
//     119,
//     120,
//     121,
//     122,
//     123,
// ]);

// function overrideKeyboardEvent(e){
//     if (e.ctrlKey || e.altKey) {
// 	if (e.keyCode != 173 && e.keyCode != 171) {
// 	    console.log("takeover for", e);
// 	    disabledEventPropagation(e);
// 	    e.preventDefault();
// 	    return false;
// 	}
//     } else if (overrideKeyCodes.has(e.keyCode)) {
// 	console.log("takeover for", e);
// 	switch(e.type){
// 	case "keydown":
// 	    if(!keyIsDown[e.keyCode]){
// 		keyIsDown[e.keyCode] = true;
// 		// do key down stuff here
// 	    }
// 	    break;
// 	case "keyup":
// 	    delete(keyIsDown[e.keyCode]);
// 	    // do key up stuff here
// 	    break;
// 	}
// 	disabledEventPropagation(e);
// 	e.preventDefault();
// 	return false;
//     } else {
// 	console.log("no takeover for", e);
//     }
// }

// function disabledEventPropagation(e){
//   if(e){
//     if(e.stopPropagation){
//       e.stopPropagation();
//     } else if(window.event){
//       window.event.cancelBubble = true;
//     }
//   }
// }
