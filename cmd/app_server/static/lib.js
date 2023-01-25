"use strict";


let LIB = {};

LIB.siblings = function (elem, includeSelf) {
    let siblings = [];

    // if no parent, return empty list
    if (!elem.parentNode) {
        return siblings;
    }

    // first child of the parent node
    let sibling = elem.parentNode.firstElementChild;

    do {

        if (includeSelf || sibling !== elem) {
            siblings.push(sibling);
	} 
    } while (sibling = sibling.nextElementSibling);

    return siblings;
}

LIB.validUUIDv4 = function (uuid) {
    let uuidRE = /^[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}$/;
    let res = uuid.match(uuidRE);
    return (res !== undefined && res !== null);
}

LIB.uuidv4 = function () {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
        var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

LIB.sleep = function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

LIB.max = function (i1, i2) {
    if (i1 > i2)
        return i1;
    return i2;
}
