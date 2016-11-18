package template

import "fmt"
import "bytes"
import "gnd.la/util/generic"

type instructions []inst

func (i instructions) replace(idx int, count int, repl []inst) []inst {
	// look for jumps before the block which need to be adjusted
	for ii, v := range i[:idx] {
		switch v.op {
		case opJMP, opJMPT, opJMPF, opNEXT:
			val := int(int32(v.val))
			if ii+val > idx {
				i[ii] = inst{v.op, valType(int32(val + len(repl) - count))}
			}
		}
	}
	// look for jumps after the block which need to be adjusted
	start := idx + count
	for ii, v := range i[start:] {
		switch v.op {
		case opJMP, opJMPT, opJMPF, opNEXT:
			val := int(int32(v.val))
			if ii+val < 0 {
				i[ii+start] = inst{v.op, valType(int32(val - len(repl) + count))}
			}
		}
	}
	var ret []inst
	ret = append(ret, i[:idx]...)
	ret = append(ret, repl...)
	ret = append(ret, i[idx+count:]...)
	return ret
}

func (i instructions) replaceTemplateInvocation(idx int, repl []inst) []inst {
	ti := i[idx]
	if ti.op != opTEMPLATE {
		panic(fmt.Errorf("OP at index %d is not opTEMPLATE", idx))
	}
	// Check if we can also remove the pipeline which provides arguments
	// to the template invocation.
	count := 1
	if idx < len(i)-1 {
		pi := i[idx+1]
		if pi.op == opPOP {
			// Remove the POP
			count++
			// This opPOP removes the arguments which were set up for
			// the template call
			if pi.val == 0 {
				// POP until the mark
				for {
					count++
					idx--
					if i[idx].op == opMARK {
						break
					}
				}
			} else {
				// Remove until we find as many pushes as this
				// pop removes
				stack := int(pi.val)
				for stack > 0 {
					count++
					idx--
					ii := i[idx]
					stack -= ii.pushes()
					stack += ii.pops()
				}
			}
		}
	}
	return i.replace(idx, count, repl)
}

// Returns true iff the program references the byte slice determined
// by val
func (p *program) referencesBytes(val valType) bool {
	for _, code := range p.code {
		for _, pi := range code {
			if pi.op == opWB && pi.val == val {
				return true
			}
		}
	}
	return false
}

// removeInternedBytes removes the internet byte slice determined
// by val, adjusting all required instructions.
func (p *program) removeInternedBytes(val valType) {
	idx := int(val)
	p.bs = append(p.bs[:idx], p.bs[idx+1:]...)
	for _, code := range p.code {
		for ii, pi := range code {
			if pi.op == opWB && pi.val >= val {
				ni := pi
				ni.val = pi.val - 1
				code[ii] = ni
			}
		}
	}
}

func (p *program) optimize() {
	p.removeEmptyTemplateInvocations()
	p.stitch()
	p.mergeWriteBytes()
}

// mergeWriteBytes merges adjacent opWB operations into a single WB
func (p *program) mergeWriteBytes() {
	removedReferences := make(map[valType]struct{})
	for name, code := range p.code {
		wb := -1
		ii := 0
		checkWBMerge := func() {
			if wb >= 0 {
				// We had a WB sequence, check its length
				// and merge it if appropriate
				count := ii - wb
				if count > 1 {
					var buf bytes.Buffer
					var refs []valType
					for c := wb; c < ii; c++ {
						wbInst := code[c]
						removedReferences[wbInst.val] = struct{}{}
						refs = append(refs, wbInst.val)
						buf.Write(p.bytesValue(wbInst.val))
					}
					repl := []inst{
						inst{op: opWB, val: p.internBytes(buf.Bytes())},
					}
					code = instructions(code).replace(wb, count, repl)
					compilerDebugf("merged %d WB instructions with byte slice refs %v at PC %d (new ref %d)\n",
						count, refs, wb, repl[0].val)
				}
			}
		}
		for ; ii < len(code); ii++ {
			v := code[ii]
			if v.op == opWB {
				if wb < 0 {
					wb = ii
				}
			} else {
				checkWBMerge()
				wb = -1
			}
		}
		checkWBMerge()
		p.code[name] = code
	}
	// Sort references from higher to lower, so we
	// don't invalidate them while they're being removed
	sortedReferences := make([]valType, 0, len(removedReferences))
	for r := range removedReferences {
		sortedReferences = append(sortedReferences, r)
	}
	generic.SortFunc(sortedReferences, func(a, b valType) bool {
		return a > b
	})
	compilerDebugln("candidates for byte slice removal", sortedReferences)
	for _, r := range sortedReferences {
		if !p.referencesBytes(r) {
			compilerDebugln("will remove byte slice reference", r)
			p.removeInternedBytes(r)
		}
	}
}

// Remove {{ template }} invocations which call into an
// empty template.
func (p *program) removeEmptyTemplateInvocations() {
	for name, code := range p.code {
		for ii := 0; ii < len(code); ii++ {
			v := code[ii]
			if v.op == opTEMPLATE {
				_, t := decodeVal(v.val)
				tmplName := p.strings[t]
				tmplCode, ok := p.code[tmplName]
				if !ok {
					panic(fmt.Errorf("missing template %q", tmplName))
				}
				if len(tmplCode) == 0 {
					// Empty template invocation, remove it
					compilerDebugf("remove empty template invocation %q from %q\n", tmplName, name)
					code = instructions(code).replaceTemplateInvocation(ii, nil)
				}
			}
		}
		p.code[name] = code
	}
}

func (p *program) stitchTree(name string) {
	// TODO: Save the name of the original template somewhere
	// so we can recover it for error messages. Until we fix
	// that problem we're only stitching trees which are just
	// a WB. In most cases, this will inline the top and bottom
	// hooks, giving already a nice performance boost.
	code := p.code[name]
	for ii := 0; ii < len(code); ii++ {
		v := code[ii]
		if v.op == opTEMPLATE {
			_, t := decodeVal(v.val)
			tmpl := p.strings[t]
			repl := p.code[tmpl]
			if len(repl) == 1 && repl[0].op == opWB {
				// replace the tree
				code = instructions(code).replaceTemplateInvocation(ii, repl)
				compilerDebugf("replaced template %q invocation from %q with WB at PC %d\n", tmpl, name, ii)
				ii--
			}
		}
	}
	p.code[name] = code
}

func (p *program) stitch() {
	p.stitchTree(p.tmpl.root)
}
