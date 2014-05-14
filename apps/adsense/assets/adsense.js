(function(name) {
    var ns = window[name] = window[name] || {};

    ns.Init = function(disabled) {
    if (ns._ready) {
        return;
    }
    ns._ready = true;
    // Check if we have any ads on the page
    if ($('.adsense-box').length == 0) {
        return;
    }
    // If someone sets this function to the handler of
    // an event, disabled will be the event, so we must
    // handle that case.
    if (disabled && !(disabled instanceof jQuery.Event)) {
        ns._HideAds();
        return
    }
    $.getScript('//pagead2.googlesyndication.com/pagead/js/adsbygoogle.js',
        function(data, textStatus, jqxhr) {
            if (jqxhr.status == 200) {
                ns._ShowAds();
            } else {
                ns._HideAds();
            }
        });
    }

    ns._ShowAds = function() {
        $('.adsense-responsive-fixed').each(function() {
            ns._ShowResponsiveFixedAd($(this));
        });
    }

    ns._HideAds = function() {
        $('.adsense-box').each(function() {
            $(this).parent().css('display', 'none');
        });
    }

    ns._ShowResponsiveFixedAd = function(ad) {
        var ins = ad.find('ins.adsbygoogle');
        var status = ins.data('adsbygoogle-status');
        if (!status) {
            // call again in a bit
            setTimeout(function() { __AdsenseShowResponsiveFixedAd(ad) }, 100);
            return;
        }
        if (status !== 'done') {
            // ad failed
            return;
        }
        var button = ad.find('.adsense-hide-button');
        var total = ad.height() + button.height();
        ad.css('bottom', -total + 'px');
        // reflow
        ad.height();
        ad.animate({'bottom': 0}, 'slow');
        var hidden = false;
        button.click(function() {
            if (hidden) {
                ad.animate({'bottom': 0}, 'slow');
                button.removeClass('adsense-out');
            } else {
                button.addClass('adsense-out');
                ad.animate({'bottom': -ad.height() + 'px'}, 'slow');
            }
            hidden = !hidden;
            return false;
        });
    }

    // If the ads have not been enabled or disabled when $(window).load
    // fires, enable them.
    $(window).load(ns.Init);
})('AdSense');
