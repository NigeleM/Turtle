best = 1e18; count = 0
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); n = 1000000; flags = Array.new(n, true); count = 0; i = 2; while i < n; if flags[i]; count += 1; j = i * i; while j < n; flags[j] = false; j += i; end; end; i += 1; end; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{count} ms=#{'%.1f' % best}"
