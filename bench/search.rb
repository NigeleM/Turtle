best = 1e18; found = 0
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); nums = Array.new(300000) { |i| i * 2 }; found = 0; x = 1; 300000.times { x = (x * 75 + 74) % 600001; lo = 0; hi = 299999; while lo <= hi; mid = (lo + hi) / 2; v = nums[mid]; if v == x then found += 1; lo = hi + 1; elsif v < x then lo = mid + 1; else hi = mid - 1; end; end }; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{found} ms=#{'%.1f' % best}"
