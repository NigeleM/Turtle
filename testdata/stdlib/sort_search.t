// sort_search.t: the sort and search libraries on lists, sets, maps,
// text, assembled values, and collections inside collections.
import sort
import search
import lib/verify [check, finish]

assemble Book [title, author, price]
books = list [Book["Dune", "Herbert", 950], Book["Emma", "Austen", 700], Book["Kindred", "Butler", 950], Book["Persuasion", "Austen", 650]]

// ---------- min_sort and max_sort, with every kind of key ----------
check["numbers", min_sort[list [3, 1.5, 2, -4]], list [-4, 1.5, 2, 3]]
check["largest first", max_sort[list [3, 1, 2]], list [3, 2, 1]]
cheap = min_sort[books, b gives price of b]
check["by a lambda", title of cheap at get[0], "Persuasion"]
check["by a field, first", title of max_sort[books, "price", "first"], "Dune"]
check["equal prices keep their order", title of max_sort[books, "price"] at get[1], "Kindred"]
two = min_sort[books, "price", 2]
check["a count", length of two, 2]
check["several keys", title of min_sort[books, list ["author", "price"]] at get[1], "Emma"]
check["nothing there", min_sort[list [], none, "first"], none]
check["the original is unchanged", title of books at get[0], "Dune"]

// ---------- lists of lists, sets, text, mixed values ----------
pairs = list [list [2, "b"], list [10, "a"], list [2, "a"]]
check["lists item by item", min_sort[pairs], list [list [2, "a"], list [2, "b"], list [10, "a"]]]
check["by position 1", min_sort[pairs, 1] at get[2], list [2, "b"]]
check["a set", min_sort[set [3, 1, 2]], list [1, 2, 3]]
check["text", min_sort["cab"], list ["a", "b", "c"]]
check["mixed kinds", min_sort[list ["b", none, 2, true]], list [none, true, 2, "b"]]
check["assembled values, no key", title of min_sort[books] at get[0], "Dune"]

// ---------- maps: by key, by value, maps of maps ----------
ages = map ["Cy": 41, "Ann": 30, "Bo": 25]
check["a map by key", min_sort[ages], map ["Ann": 30, "Bo": 25, "Cy": 41]]
check["a map by value", max_sort[ages, a gives a], map ["Cy": 41, "Ann": 30, "Bo": 25]]
check["youngest", min_sort[ages, [k, v] gives v, "first"], "Bo"]
staff = map ["x": map ["pay": 5], "y": map ["pay": 9]]
check["a map of maps", max_sort[staff, "pay", "first"], "y"]
check["reverse_list", reverse_list[ages], map ["Bo": 25, "Ann": 30, "Cy": 41]]
check["is_sorted", is_sorted[list [1, 2, 2, 5]], true]

// ---------- the classic algorithms give min_sort's answer ----------
nums = list [5, -3, 9, 0, 9, 2, -3, 7]
want = min_sort[nums]
check["bubble", bubble_sort[nums], want]
check["insertion", insertion_sort[nums], want]
check["selection", selection_sort[nums], want]
check["merge", merge_sort[nums], want]
check["quick", quick_sort[nums], want]
check["heap", heap_sort[nums], want]
check["shell", shell_sort[nums], want]
check["counting", counting_sort[nums], want]
check["radix", radix_sort[nums], want]

// ---------- finding ----------
check["find_first", title of find_first[books, b gives price of b < 900], "Emma"]
check["find_last", title of find_last[books, b gives price of b == 950], "Kindred"]
check["find_all on a map", find_all[ages, a gives a > 26], map ["Cy": 41, "Ann": 30]]
check["find_first on a map gives the key", find_first[ages, [k, v] gives v < 30], "Bo"]
check["find_key", find_key[ages, 30], "Ann"]
check["count_where", count_where[nums, n gives n > 0], 5]
check["find_index", find_index[nums, n gives n == 9], 2]
check["nothing found", find_first[nums, n gives n > 100], none]

// ---------- searching a sorted list: every algorithm agrees ----------
check["linear", linear_search[want, 9], 6]
check["binary", binary_search[want, 9], 6]
check["jump", jump_search[want, 9], 6]
check["exponential", exponential_search[want, 9], 6]
check["interpolation", interpolation_search[want, 9], 6]
check["ternary", ternary_search[want, 9], 6]
check["missing", binary_search[want, 4], -1]
check["insert_position", insert_position[want, 4], 4]
by_price = min_sort[books, "price"]
check["binary by a field", binary_search[by_price, 950, "price"], 2]

// ---------- sort is still the in-place statement too ----------
copy_of = list [3, 1, 2]
sort copy_of .
check["sort statement", copy_of, list [1, 2, 3]]
finish[]
