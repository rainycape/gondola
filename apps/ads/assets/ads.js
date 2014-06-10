(function(name) {
    var ns = window[name] = window[name] || {};
    {{ $last := add (len @providers) -1 }}
    var defaults = {
        {{ range $ii, $v := @providers }}
            {{ $v.Name }}: true{{ if neq $ii $last }},{{ end }}
        {{ end }}
    };

    var networks = {
        {{ range $ii, $v := @providers }}
            {{ $v.Name}}: {
                Name: '{{ $v.Name }}',
                Class: '{{ $v.className }}',
                Script: '{{ $v.script }}'
            }{{ if neq $ii $last }},{{ end }}
        {{ end }}
    };

    ns.Init = function(options) {
        options = $.extend({}, defaults, options);
        if (ns._ready) {
            return;
        }
        ns._ready = true;
        // Check if we have any ads on the page
        if ($('.ads-box').length == 0) {
            return;
        }
        for (var key in networks) {
            var network = networks[key];
            var selector = '.' + network.Class;
            var ads = $(selector);
            if (ads.length > 0) {
                ads.each(function () {
                    var $this = $(this);
                    if (!$this.is(':visible')) {
                        var f = ns._getNetworkFunction(network, 'remove');
                        if (f) {
                            f($this);
                        }
                        $this.remove();
                    }
                });
                ads = $(selector);
            }
            if (ads.length > 0) {
                var setup = ns._getNetworkFunction(network, 'setup');
                if (setup) {
                    ads.each(function () {
                        var ad = $(this);
                        setup(ad);
                    });
                }
                if (options[key]) {
                    if (network.Script) {
                        (function(selector) {
                            console.log('loading ', network.Script);
                            $.getScript(network.Script, function(data, textStatus, jqxhr) {
                                if (jqxhr.status == 200) {
                                    ns._ShowAds(selector);
                                } else {
                                    ns._HideAds(selector);
                                }
                            });
                        })(selector);
                    } else {
                        ns._ShowAds(selector);
                    }
                } else {
                    ns._HideAds(selector);
                }
            }
        }
    }

    ns._ShowAds = function(selector) {
        $('.ads-fixed').each(function() {
            var $this = $(this);
            if (!selector || $this.find(selector).length) {
                ns._ShowFixedAd($this);
            }
        });
    }

    ns._HideAds = function(selector) {
        $('.ads-box').each(function() {
            var $this = $(this);
            if (!selector || $this.is(selector)) {
                $this.parent().css('display', 'none');
            }
        });
    }

    ns._ShowFixedAd = function(ad) {
        var status = ns._adStatus(ad);
        if (!status) {
            // call again in a bit
            setTimeout(function() { ns._ShowFixedAd(ad) }, 100);
            return;
        }
        if (status !== 'done') {
            // ad failed
            return;
        }
        var button = ad.find('.ads-hide-button');
        button.css('display', 'block');
        var total = ad.height() + button.height();
        var prop = 'bottom';
        if (ad.hasClass('ads-fixed-top')) {
                prop = 'top';
        }
        ad.css(prop, -total + 'px');
        ad.addClass('ads-box-visible');
        // reflow
        ad.height();
        var animation = {};
        animation[prop] = 0;
        ad.animate(animation, 'slow');
        var hidden = false;
        button.click(function() {
            var pos;
            if (hidden) {
                pos = '0';
            } else {
                pos = -ad.height() + 'px';
            }
            ad.toggleClass('ads-box-visible');
            var animation = {};
            animation[prop] = pos;
            ad.animate(animation, 'slow');
            hidden = !hidden;
            return false;
        });
    }

    ns._getAdNetwork = function(ad) {
        for (var key in networks) {
            var network = networks[key];
            var selector = '.' + network.Class;
            if (ad.is(selector) || ad.find(selector).length) {
                return network;
            }
        }
        return null;
    }

    ns._getNetworkFunction = function(n, name) {
        var f = ns['_'+name+n.Name];
        if (f && f instanceof Function) {
            return f
        }
        return null
    }

    ns._adStatus = function(ad) {
        var network = ns._getAdNetwork(ad);
        if (network) {
            var f = ns._getNetworkFunction(network, 'status');
            if (f) {
                return f(ad);
            }
            return 'done';
        }
        return 'failed';
    }

    // Network specific setup and status functions

    ns._statusAdSense = function(ad) {
        return ad.find('ins.adsbygoogle').data('adsbygoogle-status');
    }

    ns._removeAdSense = function(ad) {
        // Remove ad from window.adsbygoogle
        if (window.adsbygoogle) {
            window.adsbygoogle.pop();
        }
    }

    ns._setupChitika = function(ad) {
        var div = ad.find('div').slice(0, 1);
        var publisher = div.data('publisher');
        var width = parseInt(div.data('width'), 10);
        var height = parseInt(div.data('height'), 10);
        if (!width || isNaN(width)) {
            width = ad.width();
        }
        if (!height || isNaN(height)) {
            height = ad.height();
        }
        var sid = div.data('sid');
        if (window.CHITIKA === undefined) {
            window.CHITIKA = { 'units' : [] };
        };
        var unit = {"calltype":"async[2]","publisher":publisher,"width":width,"height":height,"sid":sid};
        var placement_id = window.CHITIKA.units.length;
        window.CHITIKA.units.push(unit);
        $('<div/>').attr('id', 'chitikaAdBlock-' + placement_id).appendTo(ad);
    }

    // If the ads have not been enabled or disabled when $(window).load
    // fires, enable them.
    $(window).load(ns.Init);
})('Ads');
