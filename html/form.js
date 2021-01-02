function selectItem (selectedItem) {
	document.searchForm.itemName.value = selectedItem;
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
