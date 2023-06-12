#!/usr/bin/env ruby
# frozen_string_literal: true

require 'net/http'
require 'sensu-plugin/check/cli'

class CheckPhpFpm < Sensu::Plugin::Check::CLI
  option :hostname,
         short: '-h HOSTNAME',
         long: '--host HOSTNAME',
         description: 'Nginx hostname',
         default: '127.0.0.1'

  option :port,
         short: '-p PORT',
         long: '--port PORT',
         description: 'Nginx port',
         proc: proc(&:to_i),
         default: 80

  option :path,
         short: '-q PATH',
         long: '--path PATH',
         description: 'Path to your fpm ping',
         default: 'fpm-ping'

  option :response,
         description: 'Expected response',
         short: '-r RESPONSE',
         long: '--response RESPONSE',
         default: 'pong'

  def run
    response = Net::HTTP.start(config[:hostname], config[:port]) do |connection|
      request = Net::HTTP::Get.new("/#{config[:path]}")
      connection.request(request)
    end

    if response.code == '200'
      if response.body == config[:response]
        ok config[:response].to_s
      else
        critical "#{response.body} instead of #{config[:response]}"
      end
    else
      critical "Error, http response code: #{response.code}"
    end
  end
end
