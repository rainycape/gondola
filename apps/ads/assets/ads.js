(function(name) {
    var ns = window[name] = window[name] || {};
    var defaults = {
        AdSense: true
    };

    var networks = {
        AdSense: {
            Option: 'AdSense',
            Class: 'ads-adsense',
            Script: '//pagead2.googlesyndication.com/pagead/js/adsbygoogle.js',
            Status: function(ad) {
                return ad.find('ins.adsbygoogle').data('adsbygoogle-status');
            }
        }
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
            if ($(selector).length > 0) {
                if (options[network.Option]) {
                    if (network.Script) {
                        $.getScript(network.Script, function(data, textStatus, jqxhr) {
                            if (jqxhr.status == 200) {
                                ns._ShowAds(selector);
                            } else {
                                ns._HideAds(selector);
                            }
                        });
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
        $('.ads-responsive-fixed').each(function() {
            var $this = $(this);
            if (!selector || $this.find(selector).length) {
                ns._ShowResponsiveFixedAd($this);
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

    ns._ShowResponsiveFixedAd = function(ad) {
        var status = ns._adStatus(ad);
        if (!status) {
            // call again in a bit
            setTimeout(function() { ns._ShowResponsiveFixedAd(ad) }, 100);
            return;
        }
        if (status !== 'done') {
            // ad failed
            return;
        }
        var button = ad.find('.ads-hide-button');
        button.css('display', 'block');
        var total = ad.height() + button.height();
        ad.css('bottom', -total + 'px');
        ad.addClass('ads-box-visible');
        // reflow
        ad.height();
        ad.animate({'bottom': 0}, 'slow');
        var hidden = false;
        button.click(function() {
            var bottom;
            if (hidden) {
                bottom = '0';
            } else {
                bottom = -ad.height() + 'px';
            }
            ad.toggleClass('ads-box-visible');
            ad.animate({'bottom': bottom}, 'slow');
            hidden = !hidden;
            return false;
        });
    }

    ns._adStatus = function(ad) {
        for (var key in networks) {
            var network = networks[key];
            var selector = '.' + network.Class;
            if (ad.is(selector) || ad.find(selector).length) {
                if (network.Status) {
                    return network.Status(ad);
                }
                return 'done';
            }
        }
        return 'failed';
    }

    // If the ads have not been enabled or disabled when $(window).load
    // fires, enable them.
    $(window).load(ns.Init);
})('Ads');
