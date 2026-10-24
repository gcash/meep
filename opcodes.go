// Most of the opcode descriptions in this file are adapted from Libauth
// (https://github.com/bitauth/libauth), from
// src/lib/vm/instruction-sets/bch/chips/bch-chips-descriptions.ts at commit
// 60aec239cc2d57ae21d0069c5bbafb346abc9b66. They were changed to use bchd's
// opcode names, to drop notes about the opcode limit that the May 2025
// upgrade removed, to cover Schnorr multisig, and to fix a few details. The
// descriptions of the opcodes that the May 2026 upgrade added or enabled are
// written from bchd's documentation. Libauth is used under this license:
//
// MIT License
//
// Copyright (c) Jason Dreyzehner
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import "strings"

// unknownOpcode describes the opcodes that are not assigned an operation.
const unknownOpcode = "Error unless found in an unexecuted conditional branch."

// opcodeDescriptions describes what each opcode does, keyed by the name that
// bchd's disassembly uses. describeOpcode covers the data pushes and the
// unknown opcodes.
var opcodeDescriptions = map[string]string{
	"OP_0":                     "Push the VM Number 0 onto the stack.",
	"OP_PUSHDATA1":             "Read the next Uint8 and push that number of bytes onto the stack.",
	"OP_PUSHDATA2":             "Read the next little-endian Uint16 and push that number of bytes onto the stack.",
	"OP_PUSHDATA4":             "Read the next little-endian Uint32 and push that number of bytes onto the stack.",
	"OP_1NEGATE":               "Push the VM Number -1 onto the stack.",
	"OP_RESERVED":              "Error unless found in an unexecuted conditional branch.",
	"OP_1":                     "Push a 1 (VM Number) onto the stack.",
	"OP_2":                     "Push a 2 (VM Number) onto the stack.",
	"OP_3":                     "Push a 3 (VM Number) onto the stack.",
	"OP_4":                     "Push a 4 (VM Number) onto the stack.",
	"OP_5":                     "Push a 5 (VM Number) onto the stack.",
	"OP_6":                     "Push a 6 (VM Number) onto the stack.",
	"OP_7":                     "Push a 7 (VM Number) onto the stack.",
	"OP_8":                     "Push an 8 (VM Number) onto the stack.",
	"OP_9":                     "Push a 9 (VM Number) onto the stack.",
	"OP_10":                    "Push a 10 (VM Number) onto the stack.",
	"OP_11":                    "Push an 11 (VM Number) onto the stack.",
	"OP_12":                    "Push a 12 (VM Number) onto the stack.",
	"OP_13":                    "Push a 13 (VM Number) onto the stack.",
	"OP_14":                    "Push a 14 (VM Number) onto the stack.",
	"OP_15":                    "Push a 15 (VM Number) onto the stack.",
	"OP_16":                    "Push a 16 (VM Number) onto the stack.",
	"OP_NOP":                   "No operation.",
	"OP_VER":                   "Error unless found in an unexecuted conditional branch. (Historically, this pushed a protocol version number to the stack.)",
	"OP_IF":                    "Pop the top item from the stack. If it is not \"truthy\", skip evaluation until a matching OP_ELSE or OP_ENDIF.",
	"OP_NOTIF":                 "Pop the top item from the stack. If it is \"truthy\", skip evaluation until a matching OP_ELSE or OP_ENDIF.",
	"OP_BEGIN":                 "Begin a loop: push the position of the next instruction to the control stack, for the matching OP_UNTIL to return to.",
	"OP_UNTIL":                 "End a loop: pop the top item from the stack. If it is not \"truthy\", go back to the instruction after the matching OP_BEGIN, otherwise, continue after the OP_UNTIL. (If there is no matching OP_BEGIN, error.)",
	"OP_ELSE":                  "Invert conditional evaluation within the current OP_IF ... OP_ENDIF block. (If evaluation is enabled, disable it, if it is disabled, enable it.)",
	"OP_ENDIF":                 "End the current OP_IF/OP_NOTIF ... OP_ENDIF block.",
	"OP_VERIFY":                "Pop the top item from the stack and error if it isn't \"truthy\".",
	"OP_RETURN":                "Error when executed.",
	"OP_TOALTSTACK":            "Pop the top item from the stack and push it onto the alternate stack.",
	"OP_FROMALTSTACK":          "Pop the top item from the alternate stack and push it onto the stack.",
	"OP_2DROP":                 "Pop the top 2 items from the stack and discard them.",
	"OP_2DUP":                  "Duplicate the top 2 items on the stack. (E.g. [a, b] -> [a, b, a, b])",
	"OP_3DUP":                  "Duplicate the top 3 items on the stack. (E.g. [a, b, c] -> [a, b, c, a, b, c])",
	"OP_2OVER":                 "Duplicate the 2 items beginning at a depth of 2 on the stack. (E.g. [a, b, c, d] -> [a, b, c, d, a, b])",
	"OP_2ROT":                  "Rotate the top 6 items on the stack, bringing the fifth and sixth items to the top. (E.g. [a, b, c, d, e, f] -> [c, d, e, f, a, b])",
	"OP_2SWAP":                 "Swap the positions of the top two pairs of items on the stack. (E.g. [a, b, c, d] -> [c, d, a, b])",
	"OP_IFDUP":                 "If the top item on the stack is \"truthy\", duplicate it.",
	"OP_DEPTH":                 "Push the current number of stack items as a VM Number.",
	"OP_DROP":                  "Pop the top item from the stack and discard it. (E.g. [a] -> [])",
	"OP_DUP":                   "Duplicate the top item on the stack. (E.g. [a] -> [a, a])",
	"OP_NIP":                   "Remove the second-to-top item from the stack. (E.g. [a, b] -> [b])",
	"OP_OVER":                  "Duplicate the second-to-top item on the stack. (E.g. [a, b] -> [a, b, a])",
	"OP_PICK":                  "Pop the top item from the stack as a VM Number. Duplicate the item at that depth (zero-indexed), placing it on top of the stack. (E.g. [a, b, c, 2] -> [a, b, c, a])",
	"OP_ROLL":                  "Pop the top item from the stack as a VM Number. Move the item at that depth (zero-indexed) to the top of the stack. (E.g. [a, b, c, 2] -> [b, c, a])",
	"OP_ROT":                   "Rotate the top 3 items on the stack, bringing the third item to the top. (E.g. [a, b, c] -> [b, c, a])",
	"OP_SWAP":                  "Swap the top two items on the stack. (E.g. [a, b] -> [b, a])",
	"OP_TUCK":                  "Duplicate the item at the top of the stack, inserting it below the second-to-top item. (E.g. [a, b] -> [b, a, b])",
	"OP_CAT":                   "Pop the top 2 items from the stack and concatenate them, pushing the result.",
	"OP_SPLIT":                 "Pop the top item from the stack as an index (VM Number) and the next item as a byte array. Split the byte array into two stack items at the index (zero-based), pushing the results.",
	"OP_NUM2BIN":               "Pop the top item from the stack as an item length (VM Number) and the next item as a VM Number (without encoding restrictions). Re-encode the number using a byte array of the provided length, filling any unused bytes with zeros, then push the result. (If the requested length is too short to encode the number, error.)",
	"OP_BIN2NUM":               "Pop the top item from the stack as a VM Number without encoding restrictions. Minimally-encode the number and push the result.",
	"OP_SIZE":                  "Push the byte-length of the top stack item as a VM Number.",
	"OP_INVERT":                "Pop the top item from the stack and flip each of its bits (a bitwise NOT), pushing the result. (E.g. [0x0f] -> [0xf0])",
	"OP_AND":                   "Pop the top 2 items from the stack and perform a bitwise AND on each byte, pushing the result. If the length of the items are not equal, error.",
	"OP_OR":                    "Pop the top 2 items from the stack and perform a bitwise OR on each byte, pushing the result. If the length of the items are not equal, error.",
	"OP_XOR":                   "Pop the top 2 items from the stack and perform a bitwise XOR on each byte, pushing the result. If the length of the items are not equal, error.",
	"OP_EQUAL":                 "Pop the top two items from the stack and compare them byte-by-byte. If they are the same, push a 1 (VM Number), otherwise push a 0 (VM Number).",
	"OP_EQUALVERIFY":           "Pop the top two items from the stack and compare them byte-by-byte. If the values are different, error. (This operation is a combination of OP_EQUAL followed by OP_VERIFY.)",
	"OP_DEFINE":                "Pop the top item from the stack as a function identifier (0 to 7 bytes) and the next item as the function's bytecode. Define the function for OP_INVOKE to evaluate. (If a function with that identifier is already defined, error.)",
	"OP_INVOKE":                "Pop the top item from the stack as a function identifier. Evaluate the bytecode that OP_DEFINE defined for it, then continue after the OP_INVOKE. (If no function with that identifier is defined, error.)",
	"OP_1ADD":                  "Pop the top item from the stack as a VM Number, add 1, then push the result.",
	"OP_1SUB":                  "Pop the top item from the stack as a VM Number, subtract 1, then push the result.",
	"OP_LSHIFTNUM":             "Pop the top item from the stack as a bit count (VM Number) and the next item as a VM Number. Shift the number left by that many bits (multiplying it by 2 to the power of the bit count), then push the result.",
	"OP_RSHIFTNUM":             "Pop the top item from the stack as a bit count (VM Number) and the next item as a VM Number. Shift the number right by that many bits (dividing it by 2 to the power of the bit count, rounding down), then push the result.",
	"OP_NEGATE":                "Pop the top item from the stack as a VM Number, negate it, then push the result.",
	"OP_ABS":                   "Pop the top item from the stack as a VM Number, take its absolute value, then push the result.",
	"OP_NOT":                   "Pop the top item from the stack as a VM Number. If its value is 0, push a 1 (VM Number), otherwise, push a 0 (VM Number).",
	"OP_0NOTEQUAL":             "Pop the top item from the stack as a VM Number. If its value is not 0, push a 1 (VM Number), otherwise, push a 0 (VM Number).",
	"OP_ADD":                   "Pop the top two items from the stack as VM Numbers. Add them, then push the result.",
	"OP_SUB":                   "Pop the top two items from the stack as VM Numbers. Subtract the top item from the second item, then push the result.",
	"OP_MUL":                   "Pop the top two items from the stack as VM Numbers. Multiply them, then push the result.",
	"OP_DIV":                   "Pop the top item from the stack as a denominator (VM Number) and the next as a numerator (VM Number). Divide and push the result to the stack.",
	"OP_MOD":                   "Pop the top item from the stack as a denominator (VM Number) and the next as a numerator (VM Number). Divide and push the remainder to the stack.",
	"OP_LSHIFTBIN":             "Pop the top item from the stack as a bit count (VM Number) and the next item as a byte array. Shift the bits of the byte array left by that count, dropping the bits shifted out and filling in 0 bits from the right, then push the result. (The length does not change.)",
	"OP_RSHIFTBIN":             "Pop the top item from the stack as a bit count (VM Number) and the next item as a byte array. Shift the bits of the byte array right by that count, dropping the bits shifted out and filling in 0 bits from the left, then push the result. (The length does not change.)",
	"OP_BOOLAND":               "Pop the top two items from the stack as VM Numbers. If neither value is a 0 (VM Number), push a 1 (VM Number). Otherwise, push a 0 (VM Number).",
	"OP_BOOLOR":                "Pop the top two items from the stack as VM Numbers. If either value is not 0 (VM Number), push a 1 (VM Number). Otherwise, push a 0 (VM Number).",
	"OP_NUMEQUAL":              "Pop the top two items from the stack as VM Numbers. If the values are equal, push a 1 (VM Number). Otherwise, push a 0 (VM Number).",
	"OP_NUMEQUALVERIFY":        "Pop the top two items from the stack as VM Numbers. If the values are different, error. (This operation is a combination of OP_NUMEQUAL followed by OP_VERIFY.)",
	"OP_NUMNOTEQUAL":           "Pop the top two items from the stack as VM Numbers. If the values are not equal, push a 1 (VM Number). Otherwise, push a 0 (VM Number).",
	"OP_LESSTHAN":              "Pop the top two items from the stack as VM Numbers. If the second item is less than the top item, push a 1 (VM Number). Otherwise, push a 0 (VM Number).",
	"OP_GREATERTHAN":           "Pop the top two items from the stack as VM Numbers. If the second item is greater than the top item, push a 1 (VM Number). Otherwise, push a 0 (VM Number).",
	"OP_LESSTHANOREQUAL":       "Pop the top two items from the stack as VM Numbers. If the second item is less than or equal to the top item, push a 1 (VM Number). Otherwise, push a 0 (VM Number).",
	"OP_GREATERTHANOREQUAL":    "Pop the top two items from the stack as VM Numbers. If the second item is greater than or equal to the top item, push a 1 (VM Number). Otherwise, push a 0 (VM Number).",
	"OP_MIN":                   "Pop the top two items from the stack as VM Numbers. Push the smaller of the two numbers.",
	"OP_MAX":                   "Pop the top two items from the stack as VM Numbers. Push the larger of the two numbers.",
	"OP_WITHIN":                "Pop the top three items from the stack as VM Numbers. If the top number is within the range defined by the following two numbers (left-inclusive), push a 1 (VM Number). Otherwise, push a 0 (VM Number). (E.g. for [a, b, c]: if (b <= a), and (a < c), [1]. Else [0].)",
	"OP_RIPEMD160":             "Pop the top item from the stack and pass it through ripemd160, pushing the result onto the stack.",
	"OP_SHA1":                  "Pop the top item from the stack and pass it through sha1, pushing the result onto the stack.",
	"OP_SHA256":                "Pop the top item from the stack and pass it through sha256, pushing the result onto the stack.",
	"OP_HASH160":               "Pop the top item from the stack and pass it through sha256, then ripemd160, pushing the result onto the stack.",
	"OP_HASH256":               "Pop the top item from the stack and pass it through sha256 twice, pushing the result onto the stack.",
	"OP_CODESEPARATOR":         "Update the value of lastCodeSeparator to the instruction pointer's current value. (This reduces the coverage of signing serializations used in signature verification operations.)",
	"OP_CHECKSIG":              "Pop the top two items from the stack. Treat the top as a signature and the second as a public key. If the signature is valid, push a 1 (VM Number), otherwise push a 0 (VM Number).",
	"OP_CHECKSIGVERIFY":        "Pop the top two items from the stack. Treat the top as a signature and the second as a public key. If the signature is not valid, error. (This operation is a combination of OP_CHECKSIG followed by OP_VERIFY.)",
	"OP_CHECKMULTISIG":         "Pop items from the stack: first pop the VM Number of public keys, then pop each of those public keys. Next, pop the VM Number of required signatures, then pop each of those signatures. Finally, pop one more item: a 0 (VM Number) for ECDSA signatures, which are checked against the public keys in order, or a bitfield selecting the public keys that Schnorr signatures are checked against. If all signatures are valid (and the required number of signatures have been provided), push a 1 (VM Number), otherwise push a 0 (VM Number).",
	"OP_CHECKMULTISIGVERIFY":   "Pop items from the stack: first pop the VM Number of public keys, then pop each of those public keys. Next, pop the VM Number of required signatures, then pop each of those signatures. Finally, pop one more item: a 0 (VM Number) for ECDSA signatures, which are checked against the public keys in order, or a bitfield selecting the public keys that Schnorr signatures are checked against. If any signatures are invalid (or the required number of signatures have not been provided), error. (This operation is a combination of OP_CHECKMULTISIG followed by OP_VERIFY.)",
	"OP_NOP1":                  "No operation (reserved for future expansion).",
	"OP_CHECKLOCKTIMEVERIFY":   "Verify the transaction occurs after an absolute block time or height: read the top item on the stack as a VM Number (without removing it), and compare it to the transaction's locktime. If the required locktime has not passed, or if locktime has been disabled for this input by a maximized sequence number, error.",
	"OP_CHECKSEQUENCEVERIFY":   "Verify the transaction occurs after the output being spent has \"aged\" by a relative block time or block height since it was created: read the top item on the stack as a VM Number (without removing it), and compare it to the age encoded in the input's sequence number. If the required relative locktime has not passed, or if relative locktime has been disabled by the sequence number or the transaction version, error.",
	"OP_NOP4":                  "No operation (reserved for future expansion).",
	"OP_NOP5":                  "No operation (reserved for future expansion).",
	"OP_NOP6":                  "No operation (reserved for future expansion).",
	"OP_NOP7":                  "No operation (reserved for future expansion).",
	"OP_NOP8":                  "No operation (reserved for future expansion).",
	"OP_NOP9":                  "No operation (reserved for future expansion).",
	"OP_NOP10":                 "No operation (reserved for future expansion).",
	"OP_CHECKDATASIG":          "Pop the top 3 items from the stack. Treat the top as a public key, the second as a message, and the third as a signature. If the signature is valid, push a 1 (VM Number), otherwise push a 0 (VM Number).",
	"OP_CHECKDATASIGVERIFY":    "Pop the top 3 items from the stack. Treat the top as a public key, the second as a message, and the third as a signature. If the signature is not valid, error. (This operation is a combination of OP_CHECKDATASIG followed by OP_VERIFY.)",
	"OP_REVERSEBYTES":          "Pop the top item from the stack and reverse it, pushing the result.",
	"OP_INPUTINDEX":            "Push the index of the input being evaluated to the stack as a VM Number.",
	"OP_ACTIVEBYTECODE":        "Push the bytecode currently being evaluated, beginning after the last executed OP_CODESEPARATOR, to the stack. For Pay-to-Script-Hash (P2SH) evaluations, this is the redeem bytecode of the Unspent Transaction Output (UTXO) being spent; for all other evaluations, this is the locking bytecode of the UTXO being spent.",
	"OP_TXVERSION":             "Push the version of the current transaction to the stack as a VM Number.",
	"OP_TXINPUTCOUNT":          "Push the count of inputs in the current transaction to the stack as a VM Number.",
	"OP_TXOUTPUTCOUNT":         "Push the count of outputs in the current transaction to the stack as a VM Number.",
	"OP_TXLOCKTIME":            "Push the locktime of the current transaction to the stack as a VM Number.",
	"OP_UTXOVALUE":             "Pop the top item from the stack as an input index (VM Number). Push the value (in satoshis) of the Unspent Transaction Output (UTXO) spent by that input to the stack as a VM Number.",
	"OP_UTXOBYTECODE":          "Pop the top item from the stack as an input index (VM Number). Push the full locking bytecode of the Unspent Transaction Output (UTXO) spent by that input to the stack.",
	"OP_OUTPOINTTXHASH":        "Pop the top item from the stack as an input index (VM Number). From that input, push the outpoint transaction hash, the hash of the transaction that created the Unspent Transaction Output (UTXO) being spent, to the stack in OP_HASH256 byte order.",
	"OP_OUTPOINTINDEX":         "Pop the top item from the stack as an input index (VM Number). From that input, push the outpoint index, the index of the output in the transaction that created the Unspent Transaction Output (UTXO) being spent, to the stack as a VM Number.",
	"OP_INPUTBYTECODE":         "Pop the top item from the stack as an input index (VM Number). Push the unlocking bytecode of the input at that index to the stack.",
	"OP_INPUTSEQUENCENUMBER":   "Pop the top item from the stack as an input index (VM Number). Push the sequence number of the input at that index to the stack as a VM Number.",
	"OP_OUTPUTVALUE":           "Pop the top item from the stack as an output index (VM Number). Push the value (in satoshis) of the output at that index to the stack as a VM Number.",
	"OP_OUTPUTBYTECODE":        "Pop the top item from the stack as an output index (VM Number). Push the locking bytecode of the output at that index to the stack.",
	"OP_UTXOTOKENCATEGORY":     "Pop the top item from the stack as an input index (VM Number). If the Unspent Transaction Output (UTXO) spent by that input includes no tokens, push a 0 (VM Number) to the stack. If the UTXO does not include a non-fungible token with a capability, push the UTXO's token category, otherwise, push the concatenation of the token category and capability, where the mutable capability is represented by 1 (VM Number) and the minting capability is represented by 2 (VM Number).",
	"OP_UTXOTOKENCOMMITMENT":   "Pop the top item from the stack as an input index (VM Number). Push the token commitment of the Unspent Transaction Output (UTXO) spent by that input to the stack. If the UTXO does not include a non-fungible token, or if it includes a non-fungible token with a zero-length commitment, push a 0 (VM Number).",
	"OP_UTXOTOKENAMOUNT":       "Pop the top item from the stack as an input index (VM Number). Push the fungible token amount of the Unspent Transaction Output (UTXO) spent by that input to the stack as a VM Number. If the UTXO includes no fungible tokens, push a 0 (VM Number).",
	"OP_OUTPUTTOKENCATEGORY":   "Pop the top item from the stack as an output index (VM Number). If the output at that index includes no tokens, push a 0 (VM Number) to the stack. If the output does not include a non-fungible token with a capability, push the output's token category, otherwise, push the concatenation of the token category and capability, where the mutable capability is represented by 1 (VM Number) and the minting capability is represented by 2 (VM Number).",
	"OP_OUTPUTTOKENCOMMITMENT": "Pop the top item from the stack as an output index (VM Number). Push the token commitment of the output at that index to the stack. If the output does not include a non-fungible token, or if it includes a non-fungible token with a zero-length commitment, push a 0 (VM Number).",
	"OP_OUTPUTTOKENAMOUNT":     "Pop the top item from the stack as an output index (VM Number). Push the fungible token amount of the output at that index to the stack as a VM Number. If the output includes no fungible tokens, push a 0 (VM Number).",
	"OP_RESERVED3":             unknownOpcode,
	"OP_RESERVED4":             unknownOpcode,
	"SPECIAL_TOKEN_PREFIX":     unknownOpcode + " (This byte, 0xef, marks the token data of a transaction output.)",
	"OP_SMALLINTEGER":          unknownOpcode,
	"OP_PUBKEYS":               unknownOpcode,
	"OP_PUBKEYHASH":            unknownOpcode,
	"OP_PUBKEY":                unknownOpcode,
	"OP_INVALIDOPCODE":         unknownOpcode,
}

// describeOpcode returns what the opcode with the given name does, or "" if
// there is no such opcode.
func describeOpcode(name string) string {
	if n, ok := strings.CutPrefix(name, "OP_DATA_"); ok {
		if n == "1" {
			return "Push the next byte onto the stack."
		}
		return "Push the next " + n + " bytes onto the stack."
	}
	if strings.HasPrefix(name, "OP_UNKNOWN") {
		return unknownOpcode
	}
	return opcodeDescriptions[name]
}
