function selectItem (selectedItem) {
	document.searchForm.exactItem.value = selectedItem;
	document.searchForm.submit.click();
};

function toggleLayer (whichLayer) {
	var elem, vis;
	
	elem = document.getElementById(whichLayer);
	
	vis = elem.style;

	// if the style.display value is blank we try to figure it out here
	if (vis.display==''&&elem.offsetWidth!=undefined&&elem.offsetHeight!=undefined)
		vis.display = (elem.offsetWidth!=0&&elem.offsetHeight!=0)?'block':'none';
	
	vis.display = (vis.display==''||vis.display=='block')?'none':'block';
};

document.getElementById('chkRestricts').addEventListener('click', function(event) {
	toggleLayer('restricts');
});
document.getElementById('chkEffects').addEventListener('click', function(event) {
	toggleLayer('effects');
});
document.getElementById('chkResists').addEventListener('click', function(event) {
	toggleLayer('resists');
});
document.getElementById('chkOther').addEventListener('click', function(event) {
	toggleLayer('other');
});
document.getElementById('chkPaste').addEventListener('click', function(event) {
	toggleLayer('paste');
});
