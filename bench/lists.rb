best = 1e18; total = 0
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); nums = []; x = 1; 300000.times { x = (x * 1103515245 + 12345) % 2147483648; nums << x }; nums.sort!; total = 0; (0...300000).step(1000) { |i| total += nums[i] }; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{total} ms=#{'%.1f' % best}"
