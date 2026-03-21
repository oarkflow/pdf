package html

import (
	"fmt"
	"log"
	"strings"

	"github.com/dop251/goja"
)

// skipCDNs lists CDN hostnames whose scripts should not be fetched/executed.
var skipCDNs = []string{
	"cdn.tailwindcss.com",
	"cdn.jsdelivr.net/npm/alpinejs",
	"cdnjs.cloudflare.com/ajax/libs/jspdf",
	"cdnjs.cloudflare.com/ajax/libs/html2canvas",
}

// ExecuteScripts runs inline and external <script> elements on the DOM,
// then resolves Alpine.js bindings.
func ExecuteScripts(dom *Node, fetcher *Fetcher) error {
	vm := goja.New()

	registerDOMShim(vm, dom)

	// Collect scripts in document order
	scripts := dom.FindAll("script")
	for _, s := range scripts {
		src := s.GetAttribute("src")
		if src != "" {
			if shouldSkipCDN(src) {
				continue
			}
			data, err := fetcher.Fetch(src)
			if err != nil {
				log.Printf("jsengine: skipping script src=%s: %v", src, err)
				continue
			}
			if err := safeRunString(vm, string(data)); err != nil {
				log.Printf("jsengine: error in script src=%s: %v", src, err)
			}
			continue
		}
		// Inline script
		code := s.TextContent()
		if strings.TrimSpace(code) == "" {
			continue
		}
		if err := safeRunString(vm, code); err != nil {
			log.Printf("jsengine: error in inline script: %v", err)
		}
	}

	// Resolve Alpine.js bindings
	resolveAlpineBindings(vm, dom)

	return nil
}

func shouldSkipCDN(src string) bool {
	for _, cdn := range skipCDNs {
		if strings.Contains(src, cdn) {
			return true
		}
	}
	return false
}

// registerDOMShim registers minimal document/window/console objects on the goja runtime.
func registerDOMShim(vm *goja.Runtime, dom *Node) {
	// console stub
	console := vm.NewObject()
	noop := func(call goja.FunctionCall) goja.Value { return goja.Undefined() }
	console.Set("log", noop)
	console.Set("warn", noop)
	console.Set("error", noop)
	console.Set("info", noop)
	vm.Set("console", console)

	// window stub
	window := vm.NewObject()
	window.Set("console", console)
	vm.Set("window", window)

	// document
	doc := vm.NewObject()
	doc.Set("querySelector", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		node := querySelect(dom, sel)
		if node == nil {
			return goja.Null()
		}
		return wrapNode(vm, node)
	})
	doc.Set("querySelectorAll", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		nodes := querySelectAll(dom, sel)
		arr := vm.NewArray()
		for i, n := range nodes {
			arr.Set(fmt.Sprintf("%d", i), wrapNode(vm, n))
		}
		arr.Set("length", len(nodes))
		return arr
	})
	doc.Set("createElement", func(call goja.FunctionCall) goja.Value {
		tag := call.Argument(0).String()
		return wrapNode(vm, CreateElement(tag))
	})
	doc.Set("createTextNode", func(call goja.FunctionCall) goja.Value {
		text := call.Argument(0).String()
		return wrapNode(vm, CreateTextNode(text))
	})
	vm.Set("document", doc)
}

// wrapNode wraps a *Node as a goja object with DOM-like properties.
func wrapNode(vm *goja.Runtime, n *Node) goja.Value {
	if n == nil {
		return goja.Null()
	}
	obj := vm.NewObject()

	// textContent get/set
	obj.Set("textContent", n.TextContent())
	obj.Set("setTextContent", func(call goja.FunctionCall) goja.Value {
		n.SetTextContent(call.Argument(0).String())
		obj.Set("textContent", n.TextContent())
		return goja.Undefined()
	})

	obj.Set("getAttribute", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(n.GetAttribute(call.Argument(0).String()))
	})
	obj.Set("setAttribute", func(call goja.FunctionCall) goja.Value {
		n.SetAttribute(call.Argument(0).String(), call.Argument(1).String())
		return goja.Undefined()
	})
	obj.Set("appendChild", func(call goja.FunctionCall) goja.Value {
		// Expect a wrapped node — we can't easily unwrap, so this is a stub
		return goja.Undefined()
	})
	obj.Set("removeChild", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})

	obj.Set("id", n.ID)
	obj.Set("className", n.GetAttribute("class"))
	obj.Set("tagName", strings.ToUpper(n.Tag))

	// classList
	classList := vm.NewObject()
	classList.Set("contains", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(n.HasClass(call.Argument(0).String()))
	})
	obj.Set("classList", classList)

	// children
	childArr := vm.NewArray()
	elemIdx := 0
	for _, c := range n.Children {
		if c.IsElement() {
			childArr.Set(fmt.Sprintf("%d", elemIdx), wrapNode(vm, c))
			elemIdx++
		}
	}
	childArr.Set("length", elemIdx)
	obj.Set("children", childArr)

	// parentNode
	if n.Parent != nil {
		obj.Set("parentNode", wrapNode(vm, n.Parent))
	} else {
		obj.Set("parentNode", goja.Null())
	}

	return obj
}

// querySelect finds the first element matching a simple selector (tag, .class, #id).
func querySelect(root *Node, sel string) *Node {
	results := querySelectAll(root, sel)
	if len(results) > 0 {
		return results[0]
	}
	return nil
}

// querySelectAll finds all elements matching a simple selector.
func querySelectAll(root *Node, sel string) []*Node {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return nil
	}

	switch {
	case strings.HasPrefix(sel, "#"):
		id := sel[1:]
		return findByPredicate(root, func(n *Node) bool { return n.ID == id })
	case strings.HasPrefix(sel, "."):
		class := sel[1:]
		return findByPredicate(root, func(n *Node) bool { return n.HasClass(class) })
	default:
		// Tag selector; handle "tag.class" patterns
		if idx := strings.Index(sel, "."); idx > 0 {
			tag := sel[:idx]
			class := sel[idx+1:]
			return findByPredicate(root, func(n *Node) bool {
				return n.Tag == tag && n.HasClass(class)
			})
		}
		return root.FindAll(strings.ToLower(sel))
	}
}

func findByPredicate(root *Node, pred func(*Node) bool) []*Node {
	var results []*Node
	if root.IsElement() && pred(root) {
		results = append(results, root)
	}
	for _, c := range root.Children {
		results = append(results, findByPredicate(c, pred)...)
	}
	return results
}

// resolveAlpineBindings walks the DOM for x-data scopes and evaluates
// x-text, x-show, and x-bind bindings.
func resolveAlpineBindings(vm *goja.Runtime, dom *Node) {
	xDataNodes := findByPredicate(dom, func(n *Node) bool {
		_, ok := n.Attrs["x-data"]
		return ok
	})

	for _, scope := range xDataNodes {
		xData := scope.GetAttribute("x-data")
		if xData == "" {
			continue
		}

		// Evaluate x-data expression to get scope object
		val, err := safeRunStringVal(vm, "("+xData+")")
		if err != nil {
			log.Printf("jsengine: error evaluating x-data=%q: %v", xData, err)
			continue
		}

		scopeObj := val.ToObject(vm)
		resolveAlpineDescendants(vm, scope, scopeObj)
	}
}

func resolveAlpineDescendants(vm *goja.Runtime, node *Node, scope *goja.Object) {
	// Process x-text
	if expr, ok := node.Attrs["x-text"]; ok && expr != "" {
		result := evalInScope(vm, scope, expr)
		if result != "" {
			node.SetTextContent(result)
		}
	}

	// Process x-show
	if expr, ok := node.Attrs["x-show"]; ok && expr != "" {
		val := evalInScopeRaw(vm, scope, expr)
		if val != nil {
			if b, ok := val.Export().(bool); ok && !b {
				existing := node.GetAttribute("style")
				if existing != "" {
					node.SetAttribute("style", existing+"; display:none")
				} else {
					node.SetAttribute("style", "display:none")
				}
			}
		}
	}

	// Process x-bind:attr
	for key, expr := range node.Attrs {
		if strings.HasPrefix(key, "x-bind:") {
			attr := key[7:]
			result := evalInScope(vm, scope, expr)
			node.SetAttribute(attr, result)
		}
	}

	for _, child := range node.Children {
		if child.IsElement() {
			resolveAlpineDescendants(vm, child, scope)
		}
	}
}

func evalInScope(vm *goja.Runtime, scope *goja.Object, expr string) string {
	val := evalInScopeRaw(vm, scope, expr)
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return ""
	}
	return val.String()
}

func evalInScopeRaw(vm *goja.Runtime, scope *goja.Object, expr string) goja.Value {
	// Use with(scope) to evaluate
	vm.Set("__alpine_scope__", scope)
	code := fmt.Sprintf("(function(){ with(__alpine_scope__) { return (%s); } })()", expr)
	val, err := safeRunStringVal(vm, code)
	if err != nil {
		log.Printf("jsengine: error evaluating expression %q: %v", expr, err)
		return nil
	}
	return val
}

// safeRunString runs JS code with panic recovery, returning only an error.
func safeRunString(vm *goja.Runtime, code string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	_, err = vm.RunString(code)
	return err
}

// safeRunStringVal runs JS code with panic recovery, returning the value and error.
func safeRunStringVal(vm *goja.Runtime, code string) (val goja.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
			val = nil
		}
	}()
	val, err = vm.RunString(code)
	return val, err
}
