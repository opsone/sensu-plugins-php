#!/usr/bin/env ruby
# frozen_string_literal: true

require 'net/https'
require 'openssl'
require 'sensu-plugin/check/cli'
require 'uri'

class CheckPhpFpm < Sensu::Plugin::Check::CLI
  option :hostname,
         short: '-h HOSTNAME',
         long: '--host HOSTNAME',
         description: 'Nginx hostname',
         default: '127.0.0.1'

  option :port,
         short: '-P PORT',
         long: '--port PORT',
         description: 'Nginx port',
         proc: proc(&:to_i),
         default: 80

  option :path,
         short: '-q PATH',
         long: '--path PATH',
         description: 'Path to your fpm ping',
         default: 'fpm-ping'

  option :scheme,
         description: 'Request scheme to use',
         short: '-s SCHEME',
         long: '--scheme SCHEME',
         default: 'http://'

  option :response,
         description: 'Expected response',
         short: '-r RESPONSE',
         long: '--response RESPONSE',
         default: 'pong'

  option :ssl,
         short: '-l',
         long: '--ssl',
         boolean: true,
         description: 'Enabling SSL connections',
         default: false

  option :insecure,
         short: '-k',
         long: '--insecure',
         boolean: true,
         description: 'Enabling insecure connections',
         default: false

  def run
    url = "#{config[:scheme]}#{config[:hostname]}:#{config[:port]}/#{config[:path]}"
    uri = URI.parse(url)

    request = Net::HTTP::Get.new(uri.request_uri)
    http = Net::HTTP.new(uri.host, uri.port)

    if config[:ssl]
      http.use_ssl = true
      http.verify_mode = OpenSSL::SSL::VERIFY_NONE if config[:insecure]
    end

    response = http.request(request)

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
