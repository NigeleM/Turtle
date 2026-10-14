best = 1e18; total = 0
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); nums = []; x = 1; 300000.times { x = (x * 75 + 74) % 65537; nums << x }; nums.sort!; total = 0; 300000.times { |i| total = (total + nums[i] * (i % 7 + 1)) % 1000000007 }; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{total} ms=#{'%.1f' % best}"
