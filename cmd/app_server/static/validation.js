// config see
// package validation // import "github.com/stts-se/transtool-open/validation"

// type Config struct {
// 	PageStatusNames  string `json:"page_status_names"`
// 	StatusNames      string `json:"status_names"`
// 	ValidCharsRegexp string `json:"valid_chars_regexp"`

// 	LabelPrefix string `json:"label_prefix"`
// 	LabelSuffix string `json:"label_suffix"`
// 	Labels      string `json:"labels"`

// 	TokenSplitRegexp string `json:"token_split_regexp"`

// 	TransMustMatch    []RegexpValidation `json:"trans_must_match"`
// 	TransMustNotMatch []RegexpValidation `json:"trans_must_not_match"`
// }


// ValRes see
// package validation // import "github.com/stts-se/transtool-open/validation"

// type ValRes struct {
// 	RuleName   string `json:"rule_name"`
// 	Level      string `json:"level"`
// 	ChunkIndex int    `json:"chunk_index"`
// 	Message    string `json:"message"`
// }


// TrtValidator c.f.
// package validation // import "github.com/stts-se/transtool-open/validation"

// type Validator struct {
// 	// Has unexported fields.
// }

// func NewValidator(c Config) (Validator, error)
// func NewValidatorFromJSON(configJSON []byte) (Validator, error)
// func (v *Validator) Config() Config
// func (v *Validator) ValidateAnnotation(a protocol.AnnotationPayload) []ValRes
// func (v *Validator) ValidateTrans(t string) []ValRes
// func (v *Validator) ValidateTransChunk(c protocol.TransChunk) []ValRes


// c.f validation/validation.go
class TrtValRes {
    constructor(rule_name, level, message, chunk_index = -1) {
	this.rule_name = rule_name;
	this.level = level;
	this.message = message;
	this.chunk_index = chunk_index;
    }
}

// c.f validation/validation.go
class TrtValidator {
    constructor(config) {
	
	this.page_status_names = config.page_status_names.split(/[,\s]+/);
	this.status_names = config.status_names.split(/[,\s]+/);
	this.valid_chars_regexp = new RegExp(config.valid_chars_regexp, "gu"); // global, unicode
	this.label_prefix = config.label_prefix;
	this.label_suffix = config.label_suffix;

	// TODO change to map instead of array for faster lookup?
	this.labels = config.labels.split(/[,\s]+/);
	
	this.token_split_regexp = new RegExp(config.token_split_regexp, "u");
	this.trans_must_match = config.trans_must_match;
	this.trans_must_not_match = config.trans_must_not_match;
	//this. = config.;	
    }
    
    validateTransChars(transString) {
	let trans = transString;
	
	// Chars in labels are valid by definition
	for (let i in this.labels) {
	    trans.replaceAll(this.labels[i], "");
	};
	

	let res = [];
	
	let invalidChars = trans.replace(this.valid_chars_regexp, "");
	if (invalidChars !== "" ) {
	    let vr = new TrtValRes("invalid_chars", "error", "Invalid char(s) in transcription: '"+ invalidChars + "'");
	    res.push(vr);
	}
	
	return res;
    }

    

    // returns a list of ValRes, empty if no rules fired
    validateTrans(transString) {
	let res = [];
	for (var i in this.trans_must_match) {
	    let r =  this.trans_must_match[i];
	    if (!transString.match(r.regexp)) {
		let vr = new TrtValRes(r.rule_name, r.level, r.message); 
		res.push(vr);
	    }
	}
	
	for (var i in this.trans_must_not_match) {
	    let r =  this.trans_must_not_match[i];
	    if (transString.match(r.regexp)) {
		let vr = new TrtValRes(r.rule_name, r.level, r.message); 
		res.push(vr);
	    }
	}


	let charVres = this.validateTransChars(transString); 
	if (charVres.length > 0) {
	    res = res.concat(charVres);
	};
	
	
	// TODO Add this when there is a way of presenting non-fatal issues non-intrusively
	// let labelVres = this.validateInTransLabels(transString);
	// if (labelVres.length > 0) {
	//     res = res.concat(labelVres);
	// } 
	
	
	return res;
    }  
    
    validateInTransLabels(trans) {
	let res = [];
	let toks = trans.split(this.token_split_regexp);
	for (let i in toks ) {
	    //console.log("TOKEN", toks[i]);
	    let tok = toks[i];
	    if (tok.startsWith(this.label_prefix)  || (this.label_suffix !== "" && tok.endsWith(this.label_suffix))) {
		//console.log("LABEL", tok)
		if (!this.labels.includes(tok)) {
		    let vr = new TrtValRes("invalid_label", "error", "Invalid label: '"+ tok +"'. Valid labels: "+ this.labels.join(", "));
		    res.push(vr);
		}
	    } 
	}
    
	return res;
    }
    
}


// func validateInTransLabels(labelPrefix, labelSuffix string, tokenSplitPattern *regexp.Regexp, validLabels map[string]bool, trans string) []ValRes {

// 	var res []ValRes

// 	toks := tokenSplitPattern.Split(trans, -1)
// 	for _, t := range toks {

// 		if strings.HasPrefix(t, labelPrefix) || (labelSuffix != "" && strings.HasSuffix(t, labelSuffix)) {
// 			if !validLabels[t] {

// 				msg := fmt.Sprintf("Invalid label: '%s'", t)
// 				vr := ValRes{
// 					RuleName:   "invalid_label",
// 					Level:      "error",
// 					ChunkIndex: -1,
// 					Message:    msg,
// 				}

// 				res = append(res, vr)
// 			}
// 		}
// 	}

// 	return res
// }



// For testing in node:
// c.f validation/validation.go
//  let testConfig = `{"page_status_names":"normal delete skip","status_names":"unchecked ok ok2 skip","valid_chars_regexp":"[\\\\p{L} _.,#!?:-]","label_prefix":"#","label_suffix":"","labels":"#AGENT #CUSTOMER #OVERLAP #UNKNOWN #NOISE","token_split_regexp":"[ \\\\n,.!?]","trans_must_match":[{"rule_name":"trans_initial_label","regexp":"^\\\\s*#(AGENT|CUSTOMER|OVERLAP|UNKNOWN|NOISE)","level":"fatal","message":"Cannot OK a chunk that doesn't start with one of #AGENT, #CUSTOMER, #OVERLAP, #UNKNOWN or #NOISE"}],"trans_must_not_match":[{"rule_name":"repeated_full_stops","regexp":"[.]\\\\s*[.]","level":"error","message":"Transcription must not include repeated full stops"}]}`;

// let cfg = JSON.parse(testConfig)
// let vdator = new TrtValidator(cfg);
// //console.log(vdator.validateTrans("GG .. #AGENT JAG ÄR FELAKTIG "));
// console.log(vdator.validateTrans("GG .. #OVERALAP JAG ÄR FELAKTIG "));

//console.log(vdator.validateTrans(" &%¤#@@@ "));
//console.log(vdator.validateTrans("#AGENT APMOSTER &%¤#@@@ KWKWKWKW 99"));


