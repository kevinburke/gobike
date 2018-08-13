var plotOptions = {
  xaxis: { mode: "time" },
  legend: {position: "nw"},
  grid: {
    hoverable: true,
    borderWidth: 0,
  },
};
var color = '#002267';
var tripsPerWeek = {{ .TripsPerWeek }};
var stationsPerWeek = {{ .StationsPerWeek }};
var bikesPerWeek = {{ .BikesPerWeek }};
var tripsPerBikePerWeek = {{ .TripsPerBikePerWeek }};
var plotTooltip = function(event, pos, item) {
  if (item) {
  var x = item.datapoint[0].toFixed(0);
  var y = item.datapoint[1].toFixed(0);
    $("#tooltip").html(y + " " + item.series.label).css({top: item.pageY+5, left: item.pageX+5}).show();
  } else {
    $("#tooltip").hide();
  }
};
$.plot("#placeholder", [{color: color, data: tripsPerWeek, label: "trips"}], plotOptions);

$("<div id='tooltip'></div>").css({
  position: "absolute",
  display: "none",
  border: "1px solid #fdd",
  padding: "2px",
  "background-color": "#fee",
  opacity: 0.80,
}).appendTo("body");
$("#placeholder").bind("plothover", plotTooltip);
$.plot("#placeholder-2", [{color: color, data: stationsPerWeek, label: "stations"}], plotOptions);
$("#placeholder-2").bind("plothover", plotTooltip);
$.plot("#placeholder-3", [{color: color, data: bikesPerWeek, label: "bikes"}], plotOptions);
$("#placeholder-3").bind("plothover", plotTooltip);
$.plot("#placeholder-4", [{color: color, data: tripsPerBikePerWeek, label: "trips/bike"}], plotOptions);
$.plot("#placeholder-4", [{color: color, data: tripsPerBikePerWeek, label: "trips/bike"}], plotOptions);
$("#placeholder-4").bind("plothover", plotTooltip);
