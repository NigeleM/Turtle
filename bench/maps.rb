best = 1e18; total = 0
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); m = {}; 200000.times { |i| m["k#{i}"] = i }; total = 0; 200000.times { |i| total += m["k#{i}"] }; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{total} ms=#{'%.1f' % best}"
