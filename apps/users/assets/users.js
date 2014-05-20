(function($, name) {
  var _signin = '{{ reverse @SignIn }}';
  var _signout = '{{ reverse @SignOut }}';
  {{ if @FacebookApp }}
    var _fbAppId = '{{ @FacebookApp.Id }}';
    var _fbPerms = {{ .FacebookPermissions|json }};
    var _jsFacebookSignIn = '{{ reverse @JSSignInFacebook }}';
    var _fbChannelUrl = '{{ reverse @FacebookChannel }}';
  {{ end }}
  {{ if @GoogleApp }}
    var _googleSignIn = '{{ reverse @SignInGoogle }}';
  {{ end }}
    var _twitterSignIn = '{{ reverse @SignInTwitter }}';
    var ns = $[name] = $[name] || $({});
    ns.FB_WILL_LOAD = 'users.fb-will-load';
    ns.FB_LOADED = 'users.fb-loaded';
    ns.GOOGLE_WILL_LOAD = 'users.google-will-load';
    ns.GOOGLE_LOADED = 'users.google-loaded';
    ns.SIGNED_IN = 'users.signed-in';
    ns.SIGNED_OUT = 'users.signed-out';

    ns._defaults = {
        autofb: false, // auto-login users using Facebook
        js: true, // use JS to send forms
        popups: true, // use popups for social sign-ins
        mobilepopups: false, // use popups for social sign-ins on mobile
        modal: true // display a modal for signing in
    };

    ns._google_loaded = false;

    ns.init = function(options) {
        if (ns._initialized) {
            return;
        }
        ns._options = $.extend({}, ns._defaults, options);
        ns._initialized = true;
        if (typeof _fbAppId !== 'undefined' && _fbAppId) {
            window.fbAsyncInit = function() {
                FB.init({
                    appId: _fbAppId,
                    channelUrl: _fbChannelUrl,
                    status: true,
                    cookie: true,
                    xfbml: true
                });
                FB.Event.subscribe('auth.authResponseChange', function(response) {
                    if (ns._options.autofb) {
                        ns._onFacebookSignIn(response);
                    } else {
                        ns._checkFacebookPerms(function () {
                            ns._fb_response = response;
                        });
                    }
                });
                ns.trigger(ns.FB_LOADED);
            };
            ns.trigger(ns.FB_WILL_LOAD);
            ns._script('//connect.facebook.net/en_US/all.js');
        }
        if (typeof _googleSignIn !== 'undefined') {
            window.__usersOnGoogleSignedIn = function(resp) {
                // This function is called as soon as the button is
                // rendered if the user is already signed in. Just ignore
                // it if the user hasn't clicked the button (eventually we
                // might add an 'autogoogle' option).
                if (typeof ns._googleSigninClicked === 'undefined' || !ns._googleSigninClicked) {
                    return;
                }
                if (resp.code) {
                    $.post(_googleSignIn, 'code=' + resp.code, function(data, textStatus, jqXHR) {
                        ns._onSignedIn(data);
                    });
                }
            };
            window.___usersGoogleOnLoad = function() {
                ns._google_loaded = true;
                ns.trigger(ns.GOOGLE_LOADED);
            }
            ns.trigger(ns.GOOGLE_WILL_LOAD);
            ns._script('https://plus.google.com/js/client:plusone.js?onload=___usersGoogleOnLoad');
        }
        ns._attachEvents();
    }
    ns._script = function(src) {
        var s = document.createElement('script');
        s.type = 'text/javascript';
        s.async = true;
        s.src = src;
        var p = document.getElementsByTagName('script')[0];
        p.parentNode.insertBefore(s, p);
    }
    ns._usePopups = function() {
        return ns._options.popups && (ns._isDesktop() || ns._options.mobilepopups);
    }
    ns._attachEvents = function(doc) {
        doc = doc || $(document.body)
        if (ns._usePopups()) {
            if (typeof _fbAppId !== 'undefined' && _fbAppId) {
                doc.find('.social-sign-in a.facebook').click(function (e) {
                    e.preventDefault();
                    ns.facebookSignIn();
                });
            }

            doc.find('.social-sign-in a.twitter').click(function (e) {
                e.preventDefault();
                ns.twitterSignIn();
            });

            doc.find('.social-sign-in a.google').each(function () {
                var a = this;
                var interval = setInterval(function () {
                    if (ns._google_loaded) {
                        var options = {};
                        $(a.attributes).each(function(index, attr) {
                            if (attr.name.substr(0, 5) == 'data-') {
                                options[attr.name.substr(5)] = attr.value;
                            }
                        });
                        $(a).click(function (e) {
                            e.preventDefault();
                        });
                        $(a.parentNode).click(function () {
                            ns._googleSigninClicked = true;
                        });
                        gapi.signin.render(a.parentNode, options);
                        clearInterval(interval);
                    };
                }, 50);
            });
        }
        if (ns._options.js) {
            var wrapper = doc.find('#sign-up-form-wrapper');
            if (wrapper.length) {
                var signIn = doc.find('#sign-in-form');
                doc.find('a.go-sign-up').click(function (e) {
                    signIn.fadeOut(ns._options.speed, function() {
                        wrapper.fadeIn(ns._options.speed);
                    });
                    e.preventDefault();
                });
                doc.find('a.go-sign-in').click(function (e) {
                    wrapper.fadeOut(ns._options.speed, function() {
                        signIn.fadeIn(ns._options.speed);
                    });
                    e.preventDefault();
                });
            }
            doc.find('form .users-submit').click(ns._onFormSubmit);
            doc.find('a[href="' + _signout + '"]').click(function (e) {
                e.preventDefault();
                $.post(_signout, null, function(data, textStatus, jqXHR) {
                    var ev = $.Event(ns.SIGNED_OUT);
                    window.___user = null;
                    ns.trigger(ev);
                    if (!ev.isDefaultPrevented()) {
                        window.location.reload();
                    }
                });
            });
        }
        if (ns._options.modal) {
            doc.find('a[href="' + _signin + '"]').click(function (e) {
                e.preventDefault();
                ns.signedIn();
            });
        }
    }
    ns._onFacebookSignIn = function(response) {
        if (response.status == 'connected' && !ns.isSignedIn()) {
            var data = 'req=' + response.authResponse.signedRequest;
            $.post(_jsFacebookSignIn, data, function(data, textStatus, jqXHR) {
                ns._onSignedIn(data);
            });
        }
    }
    ns._checkFacebookPerms = function(success) {
        if (_fbPerms) {
            FB.api('/me/permissions', function(resp) {
                var perms = resp.data[0];
                for (var ii = 0; ii < _fbPerms.length; ii++) {
                    if (!perms[_fbPerms[ii]]) {
                        return
                    }
                }
                success();
            });
        } else {
            // No required permissions
            success();
        }
    }
    ns.facebookSignIn = function() {
        var opts = {};
        if (_fbPerms) {
            opts['scope'] = _fbPerms.join();
        }
        if (ns._fb_response && ns._fb_response.status == 'connected') {
            ns._onFacebookSignIn(ns._fb_response);
            ns._fb_response = undefined;
        } else {
            FB.login(ns._onFacebookSignIn, opts);
        }
    }
    ns.twitterSignIn = function () {
        var w = 600;
        var h = 510;
        var left = (screen.width/2)-(w/2);
        var top = (screen.height/2)-(h/2);
        // Use this to avoid referencing the namespace
        // from the twitter handler
        window.__users_twitter_signed_in = function(user) {
            ns._onSignedIn(user);
            delete window.__users_twitter_signed_in;
        };
        var win = window.open(_twitterSignIn + '?window=1', 'Twitter', 'height=' + h + ',width=' + w + ',menubar=0,resizable=0,toolbar=0,top=' + top + ',left=' + left);
        win.document.title = 'Connecting to Twitter...';
    }
    ns.signedIn = function(callback) {
        if (ns.isSignedIn()) {
            callback(ns.user(), false);
        } else {
            ns._callback = callback;
            var modal = $('#sign-in-modal');
            if (!modal.length) {
                $.get(_signin, 'modal=1', function (data, textStatus, jqXHR) {
                    var el = $($.trim(data));
                    ns._attachEvents(el);
                    el.css('display', 'none');
                    el.appendTo(document.body);
                    $('#sign-in-modal').modal('show');
                }, 'html');
                return;
            }
            modal.modal('show');
        }
    }
    ns.isSignedIn = function () {
        return !!ns.user();
    }
    ns.user = function () {
        if (typeof ___user !== 'undefined') {
            return ___user;
        }
        return null;
    }
    ns.errorContainer = function () {
        return $('<span class="help-block error-message error"></span>');
    }
    ns._isMobile = function () {
        return !!/Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
    }
    ns._isDesktop = function () {
        return !ns._isMobile();
    }
    ns._onSignedIn = function(user) {
        var modal = $('#sign-in-modal');
        if (modal.length) {
            modal.on('hidden.bs.modal', function () {
                modal.remove();
            });
            modal.modal('hide');
        }
        window.___user = user;
        var ev = $.Event(ns.SIGNED_IN);
        ns.trigger(ev, [user]);
        if (ns._callback) {
            ns._callback(user, true);
            ns._callback = undefined;
        } else {
            var from = ns._qs('from');
            // TODO: scheme-relative urls
            if (from && (from[0] == '/' || ns._host(from) == ns._host(window.location.href))) {
                window.location.href = from;
            } else if (!ev.isDefaultPrevented()) {
                // If for some reason we're at the sign in
                // page without a from parameter, reloading
                // it will cause a redirect to /.
                window.location.reload();
            }
        }
    }
    ns._qs = function (k) {
        var a = window.location.search.substr(1).split('&');
        for (var ii = 0; ii < a.length; ii++){
            var p = a[ii].split('=');
            if (p.length != 2) {
                continue;
            }
            if (p[0] == k) {
                return decodeURIComponent(p[1].replace(/\+/g, " "));
            }
        }
        return null;
    }
    ns._host = function(url) {
        var m = url.match(/^https?:\/\/[^/]+/);
        return m ? m[0] : null;

    }
    ns._onFormSubmit = function (e) {
        var action = form.data('js-action');
        if (!action) {
            return;
        }
        e.preventDefault();
        var button = $(this);
        var form = button.parents('form');
        form.find('.form-group').removeClass('has-error');
        form.find('.error-message').fadeOut(function () {
            $(this).remove();
        });
        button.prop('disabled', true);
        var data = form.serialize();
        $.ajax({
            type: 'POST',
            url: action,
            data: data,
            success: function(data, textStatus, jqXHR) {
                if (data.errors) {
                    for (var k in data.errors) {
                        var input = form.find('input[name=' + k + ']');
                        var div = input.parents('.form-group').first();
                        div.addClass('has-error');
                        var container = ns.errorContainer();
                        container.text(data.errors[k]);
                        container.css('display', 'none');
                        container.appendTo(div);
                        container.fadeIn();
                    }
                    button.prop('disabled', false);
                } else {
                    ns._onSignedIn(data);
                }
            },
            error: function(jqXHR, textStatus, errorThrown) {
                alert('Error: ' + textStatus);
                button.prop('disabled', false);
            }
        });
    }
})(jQuery, 'users');
