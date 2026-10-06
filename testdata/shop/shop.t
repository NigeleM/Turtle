// shop.t: a small bookshop order system that uses most of Turtle at once
// and checks its own results. Run from this folder:
//
//     echo Sam | turtle shop.t --verbose
//
// It reads data/inventory.json and data/orders.json (a list of maps),
// prices each order with coupon closures from lib/pricing.t, handles bad
// orders with safe/handle, writes a JSON report and a log, reads both
// back, cleans up, and exits 1 if any check failed.

import json
import data
import strings
import system
import time [now]
import lib/pricing
import lib/report [money, banner, title_case, count_nodes, depth]

started = now[]
failures = list []

def check[label, got, want]
    if ] got != want [
        add label to failures .
        show "  FAIL ", label, ": got ", got, ", want ", want .
    if [end]
def [end]

// ---------- setup: input, args, config ----------
cashier = ? "Cashier: "
flags = args[]
verbose = flags at find["--verbose"]
check["cashier from input", cashier, "Sam"]
check["verbose flag", verbose, true]

// There's no config file: the handler warns (to stderr) and uses defaults.
safe
    config = json_read["data/config.json"]
handle [file] e .
    if ] verbose [
        warn "no config, using defaults: ", message of e .
    if [end]
    config = map ["tax_rate": 8, "low_stock": 2]
safe [end]
check["config defaults", config at get["tax_rate"], 8]
show banner["setup"] .

// ---------- inventory: JSON -> assembled values ----------
raw = json_read["data/inventory.json"]
store = raw at get["store"]
inventory = map []
[loop][entry in raw at get["items"]]
    sku = json_get[entry, "sku"]
    t = json_get[entry, "tags"]
    tags = change t to set
    item = Item[sku, json_get[entry, "title"], json_get[entry, "price"], json_get[entry, "stock"], tags]
    inventory is inventory at add sku, item .
[loop][end]
check["inventory size", length of inventory, 4]
check["duplicate tag dropped", length of tags of item, 2]
check["category nodes", count_nodes[raw at get["categories"]], 6]
check["category depth", depth[raw at get["categories"]], 3]
check["json_get deep", json_get[raw, "categories", "children", 1, "children", 0, "name"], "go"]
check["json_get missing", json_get[raw, "categories", "children", 9, "name"], none]
show "{store}: {length of inventory} titles" .

// ---------- orders: a list of maps ----------
coupons = map ["TEN": make_discount[10], "HALF": make_discount[50]]
orders = json_read["data/orders.json"]
check["orders is a list of maps", length of orders, 7]
check["third order's customer", json_get[orders, 2, "customer"], "ann"]

results = list []
problems = map []
log_lines = list []
everyone = list []
[loop][order in orders]
    id = json_get[order, "id"]
    add json_get[order, "customer"] to everyone .
    lines = json_get[order, "lines"]
    if ] length of lines == 0 [
        add "order {id}: empty, skipped" to log_lines .
        continue
    if [end]
    safe
        // Check every line first, so a bad order changes no stock.
        subtotal = 0
        [loop][line in lines]
            item = inventory at get[json_get[line, "sku"]]
            q = json_get[line, "qty"]
            qty = change q to integer
            if ] stock of item < qty [
                fail "out of stock: {title of item}"
            if [end]
            subtotal = subtotal + price of item * qty
        [loop][end]
        total = subtotal
        coupon = json_get[order, "coupon"]
        if ] coupon != none [
            apply = coupons at get[coupon]
            total = apply[total]
        if [end]
        total = add_tax[total, config at get["tax_rate"]]
        [loop][line in lines]
            item = inventory at get[json_get[line, "sku"]]
            stock of item = stock of item - json_get[line, "qty"]
        [loop][end]
        add map ["id": id, "customer": json_get[order, "customer"], "total": total] to results .
        add "order {id}: {money[total]}" to log_lines .
    handle [key, number, custom] e .
        problems is problems at add id, kind of e .
        add "order {id}: failed ({kind of e}) {message of e}" to log_lines .
    safe [end]
[loop][end]

check["successful orders", length of results, 3]
check["problem kinds", problems, map [3: "key", 4: "custom", 5: "number"]]
check["order 1 total", json_get[results, 0, "total"], 2624]
check["order 2 total", json_get[results, 1, "total"], 3240]
check["order 6 total", json_get[results, 2, "total"], 2241]
check["log of a failure", log_lines at get[3], "order 4: failed (custom) out of stock: Gone Girl"]
check["log of a key error", log_lines at get[2], 'order 3: failed (key) key "B9" not found in map']
b1 = inventory at get["B1"]
b4 = inventory at get["B4"]
check["stock after orders", list [stock of b1, stock of b4], list [0, 5]]
show banner["orders"] .

// ---------- totals: data library, closures, module functions ----------
totals = copy[results] process r gives r at get["total"]
revenue = 0
[loop][t in totals]
    revenue = revenue + t
[loop][end]
check["revenue", revenue, 8105]
check["money", money[revenue], "$81.05"]
check["money pads cents", money[5], "$0.05"]
big = copy[totals] keep t gives t > 2300
check["keep", big, list [2624, 3240]]
check["qualified call", pricing average[totals], 2701]
check["plain call", average[totals], 2701]
check["results untouched by copy", json_get[results, 0, "id"], 1]

// An error inside lib/pricing.t names that file.
safe
    avg = average[list []]
handle [math] e .
    check["module error file", file of e, "lib/pricing.t"]
    check["module error kind", kind of e, "math"]
    check["module error text", message of e, "division by zero"]
safe [end]

// ---------- customers: sets, strings ----------
names = copy[results] process r gives r at get["customer"]
unique = change names to set
back = change unique to list
sort back .
check["unique customers", back, list ["ann", "bo"]]
pretty = copy[back] process title_case
check["title case", pretty join ", ", "Ann, Bo"]
check["everyone", length of change everyone to set, 5]

tagset = set []
[loop][sku, item in inventory]
    tagset = tagset at union[tags of item]
[loop][end]
check["all tags", length of tagset, 4]
check["superset", tagset at superset[set ["tech", "classic"]], true]

low = list []
limit = config at get["low_stock"]
[loop][sku, item in inventory]
    if ] stock of item < limit [
        add title of item to low .
    if [end]
[loop][end]
check["low stock", low, list ["Dune", "The Go Book", "Gone Girl"]]

// A quick order typed as text: split, change, sentence-style strings.
quick = "B4:2, B1:1"
qty_total = 0
[loop][part in quick at split[", "]]
    bits = part at split[":"]
    n = change bits at get[1] to integer
    qty_total = qty_total + n
[loop][end]
check["quick order", qty_total, 3]
check["find", store find "Books", 7]
first_word = store substring 0, 6
check["substring", first_word, "Turtle"]
check["replace", store at replace["Books", "Shop"], "Turtle Shop"]

// ---------- control flow ----------
i = 0
first_bad = none
[loop][i < length of log_lines]
    entry = log_lines at get[i]
    if ] "failed" isinstring entry [
        first_bad = entry
        break
    if [end]
    i = i + 1
[loop][end]
check["while + break", first_bad find "order 3", 0]

if ] revenue > 5000 [
    [if ] revenue > 10000 [
        tier = "gold"
    [else ]
        tier = "silver"
else ]
    tier = "bronze"
if [end]
check["nested if", tier, "silver"]

chart = list []
[loop][k = 0; k < length of totals; k++]
    t = totals at get[k]
    stars = ""
    [loop][s = 0; s < t div 1000; s++]
        stars = stars + "*"
    [loop][end]
    add "{k + 1} {stars}" to chart .
[loop][end]
check["chart", chart, list ["1 **", "2 ***", "3 **"]]

// ---------- the new rules ----------
safe
    bad = results + 1
handle [type] e .
    check["list + number", kind of e, "type"]
safe [end]
check["none + none", none + none, none]
greeting = load['{"store": "{store}", "empty": {}}']
check["json in single quotes", greeting at get["store"], "Turtle Books"]
check["plain braces", json_text[greeting at get["empty"]], "{}"]
check["assembled to JSON", json_text[inventory at get["B2"]], '{"sku":"B2","title":"The Go Book","price":3000,"stock":0,"tags":["tech"]}']

// ---------- report and log: write, read back, clean up ----------
report = map ["store": store, "cashier": cashier, "revenue": revenue, "tier": tier, "orders": results, "problems": problems, "low_stock": low]
json_write["shop_report.json", report]
saved = json_read["shop_report.json"]
check["report revenue", saved at get["revenue"], 8105]
check["report list of maps", json_get[saved, "orders", 2, "customer"], "bo"]
check["number keys become text", json_get[saved, "problems", "3"], "key"]

[write] shop_log.txt
"{store} log, cashier {cashier}"
[end]
[loop][line in log_lines]
    [append] shop_log.txt
    line
    [end]
[loop][end]
[read] shop_log.txt to logged [end]
check["log lines", length of logged, length of log_lines + 1]
check["log header", logged at get[0], "Turtle Books log, cashier Sam"]
check["files exist", exists["shop_report.json"] && isFile["shop_log.txt"], true]

erase["shop_report.json"]
erase["shop_log.txt"]
check["cleaned up", exists["shop_report.json"] || exists["shop_log.txt"], false]
check["clock", now[] >= started, true]

// ---------- done ----------
show banner["summary"] .
[loop][line in log_lines]
    show line .
[loop][end]
show "revenue {money[revenue]} ({tier})" .
show "failures: ", length of failures .
if ] length of failures > 0 [
    exit[1]
if [end]
