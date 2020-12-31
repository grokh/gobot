function selectItem (selectedItem) {
	document.searchForm.itemName.value = selectedItem;
	document.searchForm.submit.click();
};

function toggleLayer (whichLayer) {
	var elem, vis;
	
	if (document.getElementById) // this is the way the standards work
		elem = document.getElementById(whichLayer);
	
	vis = elem.style;
	
	vis.display = (vis.display==''||vis.display=='block')?'none':'block';
};
