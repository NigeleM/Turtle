best = 1e18; total = 0
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); parts = []; 200000.times { |i| parts << "item#{i}" }; back = parts.join(",").split(","); total = 0; back.each { |p| total += p.length }; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{total} ms=#{'%.1f' % best}"
