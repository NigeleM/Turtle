// lib/report.t: formatting helpers for shop.t.

// money shows cents as dollars: 2624 -> "$26.24", 5 -> "$0.05".
def money[cents]
    dollars = cents / 100
    rest = cents % 100
    if ] rest < 10 [
        return "${dollars}.0{rest}"
    if [end]
    return "${dollars}.{rest}"
def [end]

def banner[title]
    return "== {title at upper} =="
def [end]

// title_case: "ann" -> "Ann".
def title_case[word]
    first is word at slice 0, 1 .
    rest is word at slice 1 .
    return first at upper + rest
def [end]

// count_nodes and depth walk a JSON tree of {"name", "children"} maps
// recursively.
def count_nodes[node]
    n = 1
    kids = node at get["children"]
    [loop][child in kids]
        n = n + count_nodes[child]
    [loop][end]
    return n
def [end]

def depth[node]
    best = 0
    kids = node at get["children"]
    [loop][child in kids]
        d = depth[child]
        if ] d > best [
            best = d
        if [end]
    [loop][end]
    return best + 1
def [end]
