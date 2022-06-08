"use strict";

const baseURL =  window.location.protocol + '//' + window.location.host; // + window.location.pathname;

const keyCodeEnter = 13;
const keyCodeSpace = 32;
const keyCodeEscape = 27;

let abbrevListNames = [];

function logMessage(title, text, stacktrace) {
    if (stacktrace !== undefined) {
	//const stack = new Error().stack;
	console.log("logMessage", title, text, stacktrace.stack);
    } else {	
	console.log("logMessage", title, text);
    }
    document.getElementById("messages").textContent = title + ": " + text;    
}


function selectAbbrevList(selectListName) {
    if (selectListName !== undefined && selectListName !== null) {
	let rows = document.getElementById("abbrev_list_names").children;
	for (let i=0; i<rows.length; i++) {
	    let item = rows[i].children[0].children[0];
	    // TODO Very brittle. Make better way of getting radio elem, etc
	    let radio = item.children[0];
	    let thisName = item.children[0].value;
	    if (thisName === selectListName) {
		radio.setAttribute("checked","checked");
	    }
	}
	document.getElementById("selected_list_name").innerText = selectListName;
	document.getElementById("selected_list_name").value = selectListName;
	validateAddAbbrevButton();
	logMessage("info", "Selected abbrev list " + selectListName);
    }
}

function fetchAbbrevListNames() {
    fetch(baseURL + "/abbrev/list_lists_with_length")
	.then(response => response.json())
	.then(jzon => populateAbbrevListNames(jzon));
    
}

function populateAbbrevListNames(jzon) {
    
    let oldSelected = document.getElementById("selected_list_name").value;
    let seenOldSelected = false;
    
    let tableBody = document.getElementById("abbrev_list_names");
    tableBody.innerHTML = '';
      
    jzon.forEach(item => {
	
	let tr = document.createElement("tr");

	let radio = document.createElement("input");
	radio.setAttribute("type", "radio");
	radio.setAttribute("value", item.name);
	radio.setAttribute("name", "abbrev_list");
	if (item.name === oldSelected) {
	    radio.setAttribute("checked","checked");
	    seenOldSelected = true;
	}
	radio.addEventListener("change", function() {
	    if (this.checked) {
		let listName = this.getAttribute("value");
		loadAbbrevMap(listName);
		document.getElementById("selected_list_name").innerText = listName;
		document.getElementById("selected_list_name").value = listName;
		document.getElementById("filter_abbrev_btn").removeAttribute("disabled");
		logMessage("info", "Selected abbrev list " + listName);
	    }
	});
	let text = document.createElement("span");
	text.innerHTML = item.name + " ("+item.length+")";
	let label = document.createElement("label");
	label.appendChild(radio);	
	label.appendChild(text);
	
	let td1 = document.createElement("td");
	td1.appendChild(label);
	tr.appendChild(td1);

	let tdDel = document.createElement("td");
	tdDel.setAttribute("value", item.name);
	tdDel.setAttribute("length", item.length);
	tdDel.innerText = "×";
	tdDel.title = "Delete";
	tdDel.setAttribute("class", "red btn noborder");
	
	tdDel.addEventListener("click", function(evt) {
	    let listToDelete = this.getAttribute("value");
	    let n = this.getAttribute("length");
	    if (n > 0 ) {
		let msg = "Cannot remove non-empty list '"+ listToDelete + "' (" + n + ")"
		document.getElementById("messages").innerText = "error: " + msg;
		console.log(msg);
		return;
	    }

	    fetch(baseURL + "/abbrev/delete_list/"+ listToDelete).then(function(response) {
		if (response.ok) {
		    let msg = "Deleted list '"+ listToDelete + "'";
		    console.log(msg);
		    document.getElementById("messages").innerText =  "info: " + msg;
		    fetchAbbrevListNames();
		} else {
		    let msg = "Failed to deleted list '"+ listToDelete + "'";
		    console.log(msg);
		    console.log(response.responseText);
		    document.getElementById("messages").innerText = "error: " + msg; 
		}
	    } );
	    
	});

	tr.appendChild(tdDel);	
	tableBody.appendChild(tr);
	
    });
    
    if (!seenOldSelected) {
	document.getElementById("selected_list_name").innerText = "-";
	document.getElementById("selected_list_name").value = "";
	document.getElementById("abbrev_count").innerText = "";
	document.getElementById("filter_abbrev_btn").setAttribute("disabled","disabled");
    }
    
	
    //console.log("TABLE", table)
    
    //tableBody.appendChild(table);
    
}


function createNewListClick() {
    let name = document.getElementById("new_abbrev_list_name").value.trim();
    if (name != "") {
	createNewList(name);
    }
}

function createNewList(listName) {
    if (!listName.match(/^[a-zåäö0-9_-]+$/)) {
	logMessage("error", "Invalid abbrev list name '" + listName + "' (valid characters: 'a-zåäö0-9_-')");
	return;
    }

    fetch(baseURL + "/abbrev/create_new_list/" + listName)
	.then(async function(response){
	    if (!response.ok) {
		console.log("ERROR",response);
		throw Error(response);
            }
	    
	    let msgs = document.getElementById("messages");
	    msgs.innerText = "info: " + "Created new abbreviations list '"+ listName +"'";
	    // Empty input field
	    document.getElementById("new_abbrev_list_name").value = "";
	    fetchAbbrevListNames();
	})
    // TODO This is a bit hairy: to select a newly created list, select it, and then populate it.
    // TODO This appears to work, but there might be async things going on... need testing and/or refactoring.
	.then(selectAbbrevList(listName))
	.then(populateAbbrevsTable(listName, {}))
	.catch(function(error) {
	    console.log(error);
	    let msgs = document.getElementById("messages");
	    msgs.innerText = "error: " + "Failed to create list. Maybe there already is a list called '" + listName  +"'?";
	});
}




function loadAbbrevMap(listName) {
    
    
    let abbrevMap = {};
    // TODO: URL encode component
    fetch(baseURL + "/abbrev/list_abbrevs/"+ listName).then(async function(response){
	if (response.ok) {
	    //abbrevMap = {};
	    let abbrevs = await response.json();
	    for (let i=0; i < abbrevs.length; i++) {
		let a = abbrevs[i];
		// TODO Check for and report dupes
		abbrevMap[a.abbrev]=a.expansion;
	    };
	    
	    populateAbbrevsTable(listName, abbrevMap);
	    
	} else {
	    console.log("FAILED TO LOAD ABBREVS LIST ", listName);
	}
    });
    
}

function populateAbbrevsTable(listName, map) {
    
    let t = document.getElementById("abbrevs_table");
    t.innerHTML = '';
    
    let n = Object.keys(map).length;
    let c = document.getElementById("abbrev_count");
    c.innerText = "(" + n + ")";
    
    setAbbrevCountInListTable(listName, n);
    
    for (var k in map) {
	let tr = document.createElement("tr"); 
	let td1 = document.createElement("td");
	td1.innerText = k;
	td1.style["width"] = "50%";
	let td2 = document.createElement("td");
	td2.innerText = map[k];
	td2.style["width"] = "50%";
	let td3 = document.createElement("td");
	td3.setAttribute("value", k);
	td3.setAttribute("list_name", listName);
	td3.setAttribute("class", "red btn noborder");
	td3.innerText = "×";
	td3.title = "Delete";
	// Delete button for each abbrev
	//TODO Code below cutandpasted in other function
	td3.addEventListener("click", function() {
	    	    
	    // Name of abbrev list to delete
	    let a = this.getAttribute("value");
	    let ln = this.getAttribute("list_name");
	    
	    fetch(baseURL + "/abbrev/delete/"+ ln + "/" + a).then(function(response) {
		if(response.ok) {
		    let abbrevs = document.getElementById("abbrevs_table").children;
		    for (var i = 0; i < abbrevs.length; i++) {
			let abr = abbrevs[i].children[0];
			if (abr.innerText.trim() === a) {
			    abbrevs[i].remove();
			    document.getElementById("messages").innerText = "info: " + "Deleted '"+ a +"'";
			} 
		    }
		    // Reload abbrevs after delete?
		    // Or remove locally?
		    loadAbbrevMap(ln);
		    setAbbrevCountInListTable(ln, abbrevs.length);
		}
	    });
	    
	    
	});
	
	
	tr.appendChild(td1);
	tr.appendChild(td2);
	tr.appendChild(td3);
	
	t.appendChild(tr);
    }
    
}


function setAbbrevCountInListTable(listName, n) {
    //console.log("setAbbrevCountInListTable", listName, n);
    var rows = document.getElementById("abbrev_list_names").children;
    for (let i=0; i<rows.length; i++) {
	// TODO Very brittle. Make better way of getting radio elem, etc
	let eles = rows[i].children[0].children[0].children;
	let thisName = eles[0].value;
	let html = eles[1];
	if (thisName === listName) {
	    let radio = rows[i].children[1];
	    radio.setAttribute("length", n);
	    html.innerHTML = listName + " (" + n + ")";
	}
    }
    document.getElementById("abbrev_count").innerText = "(" + n + ")";
}

function abbrevFilter() {
    // let s = this.value.trim();
    // let id = this.getAttribute("id");

    let sa = document.getElementById("abbrev_filter").value.trim();
    let se = document.getElementById("expansion_filter").value.trim();

    // Loop over all abbrevs
    let abbrevs = document.getElementById("abbrevs_table").children;
    let n = 0;
    console.log("abbrevFilter started " + new Date());
    
    for (var i=0; i < abbrevs.length; i++) {
	let a = abbrevs[i].children[0].innerText;
	let e = abbrevs[i].children[1].innerText;
	if (sa !== "" && !a.includes(sa))
	    abbrevs[i].style.display="none";
	else if  (se !== "" && !e.includes(se)) 
	    abbrevs[i].style.display="none";
	else {
	    abbrevs[i].style["display"] = "";
	    n++;
	}
    }
    logMessage("info", "Search completed (" + n + ")");
    console.log("abbrevFilter completed " + new Date() + " for sa '" + sa + "', se '" + se + "'");
}

document.getElementById("new_abbrev").addEventListener("keypress", function(evt) {
    if (evt.keyCode == keyCodeEnter) {
	document.getElementById("add_abbrev_btn").click();
    }        
});
document.getElementById("new_expansion").addEventListener("keypress", function(evt) {
    if (evt.keyCode == keyCodeEnter) {
	document.getElementById("add_abbrev_btn").click();
    }        
});
document.getElementById("new_abbrev_list_name").addEventListener("keypress", function(evt) {
    if (evt.keyCode == keyCodeEnter) {
	document.getElementById("create_new_abbrev_list_name").click();
    }        
});

document.getElementById("new_abbrev").addEventListener("keyup", validateAddAbbrevButton);
document.getElementById("new_expansion").addEventListener("keyup", validateAddAbbrevButton);
document.getElementById("new_abbrev").addEventListener("keypress", validateAddAbbrevButton);
document.getElementById("new_expansion").addEventListener("keypress", validateAddAbbrevButton);
document.getElementById("new_abbrev").addEventListener("click", validateAddAbbrevButton);
document.getElementById("new_expansion").addEventListener("click", validateAddAbbrevButton);


function nonEmptyString(s) {
    return s !== undefined && s !== null && s.trim().length > 0;
}

function validateAddAbbrevButton() {
    let valid = true;
    if (!nonEmptyString(document.getElementById("selected_list_name").value)) {
	valid = false;
    }
    if (document.getElementById("new_abbrev").value === "" || document.getElementById("new_expansion").value === "") {
	valid = false;
    }
    if (valid) {
	document.getElementById("add_abbrev_btn").removeAttribute("disabled");
    } else {
	document.getElementById("add_abbrev_btn").setAttribute("disabled","disabled");
    }
}

function addNewAbbrev(evt) {
    let a  = document.getElementById("new_abbrev").value.trim();
    let e  = document.getElementById("new_expansion").value.trim();
    
    if (a === "") {
	logMessage("error", "Empty abbrev field");
	return;
    }
    if (!a.match(/^[a-zåäöæøA-ZÅÄÖÆØ0-9_?-]+$/)) {
	logMessage("error", "Invalid abbrev field '" + a + "' (valid characters: 'a-zåäöæøA-ZÅÄÖÆØ0-9_?-')");
	return;
    }
    if (!e.match(/^[a-zåäöæøA-ZÅÄÖÆØ0-9_ \[\]?:#-]+$/)) {
	logMessage("error", "Invalid expansion field '" + e + "' (valid characters: 'a-zåäöæøA-ZÅÄÖÆØ0-9_ \[\]?:#-')");
	return;
    }
    
    var selectedList = document.getElementById("selected_list_name").value;
    // var listNames = document.getElementById("abbrev_list_names").children;
    // for (var i = 0; i < listNames.length; i++) {

    // 	console.log("??",listNames[i].children[0].children[0]);
    // 	// TODO Very brittle. Make better way of getting radio elem, etc
    // 	if (listNames[i].children[0].children[0].type === 'radio' && listNames[i].children[0].children[0].checked) {
    //         // get value, set checked flag or do whatever you need to
    //         selectedList = listNames[i].children[0].children[0].value.trim();       
    // 	}
    // }
    
    if (selectedList === "") {
	// TODO: error message
	console.log("Cannot add new abbrev before selecting a list");
	return;
    }

    //HB 0729
    //encode e to allow for # character
    e = encodeURIComponent(e);

    
    fetch(baseURL + "/abbrev/add/" + selectedList + "/" + a + "/" + e).then(async function(response) {
	if (response.ok) {
	    // TODO: refactor with chain of then's 
	    let jzon = await response.json();
	    if (jzon.label === "error") {
	    	logMessage("error", jzon.content);
	    	return;
	    } else if (jzon.label === "info") {
	    	logMessage("info", jzon.content);
		// TODO Very brittle if table layout is changed!
		let tr = document.createElement("tr");
		let td1 = document.createElement("td");
		td1.style["width"] = "50%";
		let td2 = document.createElement("td");
		td2.style["width"] = "50%";
		let td3 = document.createElement("td");
		
		td1.innerText = a;
		td2.innerText = e;
		
		td3.innerText = "×";
		td3.setAttribute("value", a);
		td3.setAttribute("list_name", selectedList);
		td3.setAttribute("class", "red btn noborder");
		//TODO Cutandpaste from other function
		// Create a stand-alone function
		td3.addEventListener("click", function() {
		    
		    
		    // Name of abbrev list to delete
		    let a = this.getAttribute("value");
		    let ln = this.getAttribute("list_name");
		    
		    fetch(baseURL + "/abbrev/delete/"+ ln + "/" + a).then(function(response) {
			if(response.ok) {
			    let abbrevs = document.getElementById("abbrevs_table").children;
			    for (var i = 0; i < abbrevs.length; i++) {
				let abr = abbrevs[i].children[0];
				if (abr.innerText.trim() === a) {
				    abbrevs[i].remove();
				    document.getElementById("messages").innerText = "info :" + "Deleted '"+ a +"'";
				} 
			    }
			    // Reload abbrevs after delete?
			    // Or remove locally?
			    loadAbbrevMap(ln);
			}
		    });
		    
		});
		
		
		
		tr.appendChild(td1);
		tr.appendChild(td2);
		tr.appendChild(td3);
		
		document.getElementById("abbrevs_table").prepend(tr)
		let n = document.getElementById("abbrevs_table").children.length;
		setAbbrevCountInListTable(selectedList, n);
		//logMessage("info", "Added abbrevation " + a + " => " + e);
	    }
	} else {
	    logMessage("error", "Failed to add abbrevation " + a + " => " + e);
	}
    });
}






window.onload = function() {
    
        
    // Show names of existing abbrev lists
    fetchAbbrevListNames();
    
    
    let newAbbrevListBtn = document.getElementById("create_new_abbrev_list_name");
    newAbbrevListBtn.addEventListener("click", createNewListClick);
    
    document.getElementById("abbrev_filter").addEventListener("keypress", function(evt){
	if (evt.keyCode == keyCodeEnter) document.getElementById("filter_abbrev_btn").click();
    });
    document.getElementById("expansion_filter").addEventListener("keypress", function(evt) {
	if (evt.keyCode == keyCodeEnter) document.getElementById("filter_abbrev_btn").click();
    });
    document.getElementById("filter_abbrev_btn").addEventListener("click", abbrevFilter);
    
    document.getElementById("add_abbrev_btn").addEventListener("click", addNewAbbrev);

    // set width in table matching the width in the filter area
    // document.getElementById("abbrev_list_h_abbrev").setAttribute("width", document.getElementById("abbrev_filter").offsetWidth);
    // document.getElementById("abbrev_list_h_expansion").setAttribute("width", document.getElementById("expansion_filter").offsetWidth);

};
