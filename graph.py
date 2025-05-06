import os

website="https://pasgal-bs.cs.ucr.edu/bin/"

CURRENT_DIR = os.getcwd()
GRAPH_DIR = os.path.join(CURRENT_DIR, "data", "graphs")

graphs = [
"Epinions1",
"Slashdot",
"DBLP",
"com-youtube",
"skitter",
"in_2004",
"soc-LiveJournal1",
"hollywood_2009",
"com-orkut"
]

graph_map={
"Epinions1": "EP",
"Slashdot":"SLDT",
"DBLP":"DBLP",
"com-youtube":"YT",
"skitter":"SK",
"in_2004":"IN04",
"soc-LiveJournal1":"LJ",
"hollywood_2009":"HW",
"com-orkut":"OK"
}