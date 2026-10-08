require "json"
best = 1e18; total = 0
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); records = Array.new(20000) { |i| { "id" => i, "name" => "item#{i}", "price" => i % 100, "tags" => ["a", "b"] } }; back = JSON.parse(JSON.generate(records)); total = back.sum { |x| x["price"] }; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{total} ms=#{'%.1f' % best}"
