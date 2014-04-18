;// timeago initialization, requires reasonable value

$( function ()
{
	$( ".timeago" ).each( function ()
	{
		if ( ( new Date( $( this ).attr( "title" ) ) ).getFullYear() > 2000 )
		{
			$( this ).timeago();
		}
		else
		{
			$( this ).text( "Pending" );
		}
	} );
} );
