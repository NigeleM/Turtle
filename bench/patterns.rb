best = 1e18; total = 0
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); text = Array.new(50000) { |i| "id=#{i} name=item#{i} qty=#{i % 7}" }.join("\n"); total = text.scan(/qty=(\d+)/).sum { |m| m[0].to_i }; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{total} ms=#{'%.1f' % best}"
