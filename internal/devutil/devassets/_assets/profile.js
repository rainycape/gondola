function ___gondolaShowProfileInfo() {
    if (___gondolaHideProfileInfo()) {
        return;
    }
    var wrapper = document.createElement('div');
    wrapper.style.opacity = 0;
    wrapper.id = 'gondola_profile_info_iframe_wrapper';
    var iframe = document.createElement('iframe');
    iframe.style.opacity = 0;
    iframe.border = 0;
    iframe.frameBorder = 0;
    iframe.id = 'gondola_profile_info_iframe';
    iframe.name = iframe.id;
    var form = document.createElement('form');
    form.id = 'gondola_profile_info_form';
    form.action = '/debug/profile';
    form.method = 'POST';
    form.target = iframe.name;
    var input = document.createElement('input');
    input.type = 'text';
    input.name = 'data';
    input.value = ___gondola_profile_info;
    form.appendChild(input);
    wrapper.appendChild(iframe)
    document.body.appendChild(form);
    document.body.appendChild(wrapper);
    iframe.onload = function() {
        iframe.style.height = iframe.contentWindow.document.body.scrollHeight + 'px';
        iframe.style.opacity = 1;
    }
    form.submit();
    form.parentNode.removeChild(form);
    wrapper.style.opacity = 1;
}

function ___gondolaHideProfileInfo() {
    var el = document.getElementById('gondola_profile_info_iframe_wrapper');
    if (el) {
        el.parentNode.removeChild(el);
        return true;
    }
    return false;
}
