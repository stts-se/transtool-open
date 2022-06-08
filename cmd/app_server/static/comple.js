'use strict';

class Node {
    constructor() { 
	this.char = '';
	this.daus = {};
	this.leaf = false;
	this.value = -1;
    }
}


class Trie {
    constructor() {
	this.t = new Node();
	this.expandAllAccumulator = [];	
    };
    

    addWord(w, freq) {
	this.addWord0(w, freq, this.t);
    };      
    
    
    addWord0(w, freq, tree) {

	//console.log("AW0", w);
	
	if (w.lenght === 0) {
	    return;
	};
	
	
	let c = w[0];
 	let rest = w.substring(1);
	
	// TODO Why is this needed?
	if (tree.daus === undefined) {
	    tree.daus = {};
	} 
	
	if (tree.daus.hasOwnProperty(c) && rest !== '') {
	    this.addWord0(rest, freq, tree.daus[c]);
	} else {

	    let n = new Node();
	    n.char = c;
	    if (rest === '') {
		n.leaf = true;
		n.value = freq;
	    }
	    tree.daus[c] = n;
	    if (rest !== '') {

		//console.log("rest", rest);
		
		this.addWord0(rest, freq, n);
	    }
	}
	return;
    };

    lookup(w) {
	return this.lookup0(w, this.t);
    };
    
    lookup0(w, tree) {
	if (w.length === 0) {
	    return -1;
	};
	if (!tree.daus.hasOwnProperty(w[0])) {
	    return -1
	};
	
	let rest = w.substring(1);
	let node = tree.daus[w[0]];
	if (rest === '' && node.leaf) {
	    return node.value;
	};
	
	return this.lookup0(rest, node);	
    };
    
    
    expandPrefix(w){
	return this.expandPrefix0(w, w, '', this.t);
    };
    
    expandPrefix0(w, prefix, suffix, tree) {
	let c = w[0];
	let rest = w.substring(1);
	
	if (w === '') {
	    this.expandAllAccumulator = []; 
	    return this.expandAll(tree.daus, prefix, suffix);
	}
	
	
	if (!tree.daus.hasOwnProperty(c)) {
	    return [];
	};
	
	
	return this.expandPrefix0(rest, prefix, suffix, tree.daus[c]);
	
    };
    
    expandAll(tree, prefix, suffix) {
	
	if (tree.length === 0) {
	    return expandAllAccumulator;
	}
	
	for(var c in tree) {
	    if (tree[c].leaf) {		
		this.expandAllAccumulator.push({"w":prefix+c, "suff": suffix+c, "f":tree[c].value});
		this.expandAll(tree[c].daus, prefix+c, suffix+c);
	    }  else { 
		this.expandAll(tree[c].daus, prefix+c, suffix+c);
	    }
	}
	return this.expandAllAccumulator;
    };
    
}





let t = new Trie();

// t.addWord("apnos", 89);
// t.addWord("apnosar", 98);

// var w = -1;
// var g = t.lookup("öäöäöööxxx");
// if (g !== w)
//     throw "wanted "+ w+ " got "  + g;


// w = 89;
// g = t.lookup("apnos");
// if (g !== w)
//     throw "wanted "+ w+ " got "  + g;

// w = 98;
// g = t.lookup("apnosar");
// if (g !== w)
//     throw "wanted "+ w+ " got "  + g;

// w = 2;
// g = t.expandPrefix("apno").length
// if (g !== w)
//     throw "wanted "+ w+ " got "  + g;

// //console.log(t.expandPrefix("apno"));


const fs = require('fs')

try {
    const lines = fs.readFileSync('sv_SE.dict', 'utf8').toString().split("\n");
    //console.log(data)
    
    for (var i in lines) { 
	//console.log(l);
	t.addWord(lines[i], 1);

    }
} catch (err) {
    console.error("err", err)
}

console.log(t.expandPrefix(process.argv[2]));

//console.log(t.expandPrefix("strim"));
